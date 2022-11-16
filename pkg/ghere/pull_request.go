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
// call.
func (pr *PullRequest) GetNumber() int {
	return pr.PullRequest.GetNumber()
}

// GetPath returns the path of this pull request's folder relative to the
// repository's pull requests folder.
func (pr *PullRequest) GetPath(prsPath string) string {
	return filepath.Join(prsPath, fmt.Sprintf("%.6d", pr.GetNumber()))
}

func (pr *PullRequest) GetDetailsPath(prsPath string) string {
	return filepath.Join(pr.GetPath(prsPath), DETAIL_FILENAME)
}

func (pr *PullRequest) GetReviewsPath(prsPath string) string {
	return filepath.Join(pr.GetPath(prsPath), "reviews")
}

type pullRequestsFetcher struct {
	rootPath string
	repo     *Repository
}

func newPullRequestsFetcher(rootPath string, repo *Repository) *pullRequestsFetcher {
	return &pullRequestsFetcher{
		rootPath: rootPath,
		repo:     repo,
	}
}

func (pf *pullRequestsFetcher) fetch(ctx context.Context, cfg *FetchConfig, log Logger) ([]fetcher, error) {
	log.Info("Fetching pull requests for repository", "repo", pf.repo.String())
	pattern := filepath.Join(pf.repo.GetPullRequestsPath(pf.rootPath), "*", DETAIL_FILENAME)
	startPage, err := paginatedItemsStartPage(pattern, func(fn string) (bool, error) {
		pr := &PullRequest{}
		if err := readJSONFile(fn, pr); err != nil {
			return false, err
		}
		return pf.repo.Repository.GetUpdatedAt().After(pr.LastDetailFetch), nil
	})
	if err != nil {
		return nil, err
	}
	err = rateLimitedPaginated(startPage, log, func(pg int) (res *github.Response, done bool, err error) {
		var pulls []*github.PullRequest
		log.Info("Fetching page of pull requests", "page", pg)
		pulls, res, err = cfg.Client.PullRequests.List(ctx, pf.repo.GetOwner(), pf.repo.GetName(), &github.PullRequestListOptions{
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
		for _, pull := range pulls {
			pr := &PullRequest{
				PullRequest: pull,
			}
			prPath := filepath.Join(pr.GetDetailsPath(pf.rootPath), DETAIL_FILENAME)
			if err = readJSONFileOrEmpty(prPath, pr); err != nil {
				return
			}
			pr.PullRequest = pull
			pr.LastDetailFetch = time.Now()
			if err = writeJSONFile(prPath, pr, cfg.PrettyJSON); err != nil {
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
	log.Debug("Computing which pull requests' reviews and comments should be fetched", "repo", pf.repo.String())
	// Pull requests whose reviews and comments must be fetched.
	fetchReviews := []*PullRequest{}
	fetchComments := []*PullRequest{}
	prsPath := pf.repo.GetPullRequestsPath(pf.rootPath)
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
		fetchers = append(fetchers, newPullRequestReviewsFetcher(prsPath, pf.repo, fetchReviews))
	}
	if len(fetchComments) > 0 {
		fetchers = append(fetchers, newPullRequestCommentsFetcher(prsPath, pf.repo, fetchComments))
	}

	return fetchers, nil
}
