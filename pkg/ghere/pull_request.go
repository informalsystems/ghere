package ghere

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	"github.com/google/go-github/v48/github"
)

type PullRequest struct {
	PullRequest *github.PullRequest `json:"pull_request"`

	LastDetailFetch   time.Time `json:"last_detail_fetch"`
	LastReviewsFetch  time.Time `json:"last_reviews_fetch"`
	LastCommentsFetch time.Time `json:"last_comments_fetch"`
}

// GetNumber is a shortcut for accessing the inner `PullRequest.GetNumber()`
// method.
func (pr *PullRequest) GetNumber() int {
	return pr.PullRequest.GetNumber()
}

// pullRequestsFetcher fetches pull requests for a specific repository.
type pullRequestsFetcher struct {
	rootPath string
	repo     *Repository
}

var _ fetcher = (*pullRequestsFetcher)(nil)

func newPullRequestsFetcher(rootPath string, repo *Repository) *pullRequestsFetcher {
	return &pullRequestsFetcher{
		rootPath: rootPath,
		repo:     repo,
	}
}

func (pf *pullRequestsFetcher) fetch(ctx context.Context, cfg *FetchConfig, log Logger) ([]fetcher, error) {
	log.Info("Fetching pull requests for repository", "repo", pf.repo.String())
	prsPath := repoPullRequestsPath(pf.rootPath, pf.repo.GetOwner(), pf.repo.GetName())
	pattern := filepath.Join(prsPath, "*", DETAIL_FILENAME)
	startPage, err := paginatedItemsStartPage(pattern, func(fn string) (bool, error) {
		pr := &PullRequest{}
		if err := readJSONFile(fn, pr); err != nil {
			return false, fmt.Errorf("failed to read pull request detail file %s: %v", fn, err)
		}
		repoUpdatedRecently := pf.repo.Repository.GetUpdatedAt().After(pr.LastDetailFetch)
		fetchedMoreThan24hAgo := time.Since(pr.LastDetailFetch) > 24*time.Hour
		return repoUpdatedRecently && fetchedMoreThan24hAgo, nil
	})
	if err != nil {
		return nil, err
	}
	err = rateLimitedPaginated(
		ctx,
		startPage,
		cfg.RequestRetries,
		cfg.RequestTimeout,
		log,
		func(cx context.Context, pg int) (res *github.Response, done bool, err error) {
			var pulls []*github.PullRequest
			log.Info("Fetching page of pull requests", "repo", pf.repo.String(), "page", pg)
			pulls, res, err = cfg.Client.PullRequests.List(cx, pf.repo.GetOwner(), pf.repo.GetName(), &github.PullRequestListOptions{
				State:     "all",
				Sort:      "created",
				Direction: "asc",
				ListOptions: github.ListOptions{
					Page:    pg,
					PerPage: DEFAULT_PER_PAGE,
				},
			})
			if err != nil {
				return
			}
			for _, ghPull := range pulls {
				pull := &PullRequest{}
				prPath := pullRequestDetailPath(pf.rootPath, pf.repo.GetOwner(), pf.repo.GetName(), ghPull.GetNumber())
				if err = readJSONFileOrEmpty(prPath, pull); err != nil {
					return
				}
				pull.PullRequest = ghPull
				pull.LastDetailFetch = time.Now()
				if err = writeJSONFile(prPath, pull, cfg.PrettyJSON); err != nil {
					return
				}
			}
			done = len(pulls) < DEFAULT_PER_PAGE
			return
		})
	if err != nil {
		return nil, err
	}
	log.Info("Fetched all pull requests' details", "repo", pf.repo.String())

	return pf.makeReviewsAndCommentsFetchers(log)
}

func (pf *pullRequestsFetcher) makeReviewsAndCommentsFetchers(log Logger) ([]fetcher, error) {
	log.Info("Computing which pull requests' reviews and comments should be fetched", "repo", pf.repo.String())
	// Pull requests whose reviews and comments must be fetched.
	fetchReviews := []*PullRequest{}
	fetchComments := []*PullRequest{}
	prsPath := repoPullRequestsPath(pf.rootPath, pf.repo.GetOwner(), pf.repo.GetName())
	pattern := filepath.Join(prsPath, "*", DETAIL_FILENAME)
	pullRequestDetailsFiles, err := filepath.Glob(pattern)
	if err != nil {
		return nil, fmt.Errorf("failed to list pull requests' detail files from pattern %s: %v", pattern, err)
	}

	for _, fn := range pullRequestDetailsFiles {
		pr := &PullRequest{}
		if err := readJSONFile(fn, pr); err != nil {
			return nil, fmt.Errorf("failed to read pull request detail file %s: %v", fn, err)
		}
		if pr.PullRequest.GetUpdatedAt().After(pr.LastReviewsFetch) {
			fetchReviews = append(fetchReviews, pr)
		}
		if pr.PullRequest.GetUpdatedAt().After(pr.LastCommentsFetch) {
			fetchComments = append(fetchComments, pr)
		}
	}

	fetchers := []fetcher{}
	if len(fetchReviews) > 0 {
		fetchers = append(fetchers, newPullRequestReviewsFetcher(pf.rootPath, pf.repo, fetchReviews))
	}
	if len(fetchComments) > 0 {
		fetchers = append(fetchers, newPullRequestCommentsFetcher(pf.rootPath, pf.repo, fetchComments))
	}

	return fetchers, nil
}