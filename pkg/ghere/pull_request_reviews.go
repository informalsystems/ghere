package ghere

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	"github.com/google/go-github/v48/github"
)

type PullRequestReview struct {
	Review *github.PullRequestReview `json:"review"`

	PullRequestNumber int       `json:"pull_request_number"`
	LastDetailFetch   time.Time `json:"last_detail_fetch"`
	LastCommentsFetch time.Time `json:"last_comments_fetch"`
}

func LoadPullRequestReview(rootPath string, repo *Repository, prNum int, reviewID int64, mustExist bool) (*PullRequestReview, error) {
	path := pullRequestReviewDetailPath(rootPath, repo.GetOwner(), repo.GetName(), prNum, reviewID)
	review, err := LoadPullRequestReviewDirect(path, mustExist)
	if err != nil {
		return nil, err
	}
	review.PullRequestNumber = prNum
	return review, nil
}

func LoadPullRequestReviewDirect(path string, mustExist bool) (*PullRequestReview, error) {
	var err error
	review := &PullRequestReview{}
	if mustExist {
		err = readJSONFile(path, review)
	} else {
		err = readJSONFileOrEmpty(path, review)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to read pull request review detail file: %v", err)
	}
	return review, nil
}

func (r *PullRequestReview) Save(rootPath string, repo *Repository, prettyJSON bool) error {
	path := pullRequestReviewDetailPath(rootPath, repo.GetOwner(), repo.GetName(), r.PullRequestNumber, r.Review.GetID())
	if err := writeJSONFile(path, r, prettyJSON); err != nil {
		return fmt.Errorf("failed to write pull request review detail file: %v", err)
	}
	return nil
}

// pullRequestReviewsFetcher fetches reviews for a number of PRs.
type pullRequestReviewsFetcher struct {
	rootPath     string
	repo         *Repository
	pullRequests []*PullRequest
}

func newPullRequestReviewsFetcher(rootPath string, repo *Repository, pullRequests []*PullRequest) *pullRequestReviewsFetcher {
	return &pullRequestReviewsFetcher{
		rootPath:     rootPath,
		repo:         repo,
		pullRequests: pullRequests,
	}
}

func (rf *pullRequestReviewsFetcher) fetch(ctx context.Context, cfg *FetchConfig, log Logger) ([]fetcher, error) {
	for _, pr := range rf.pullRequests {
		log.Info("Fetching pull request reviews", "repo", rf.repo.String(), "pr", pr.GetNumber())
		prReviewsPath := pullRequestReviewsPath(rf.rootPath, rf.repo.GetOwner(), rf.repo.GetName(), pr.GetNumber())
		pattern := filepath.Join(prReviewsPath, "*", DETAIL_FILENAME)
		startPage, err := paginatedItemsStartPage(pattern, func(fn string) (bool, error) {
			review, err := LoadPullRequestReviewDirect(fn, true)
			if err != nil {
				return false, err
			}
			// If the pull request was last updated after this review was last
			// fetched, consider it outdated.
			return pr.PullRequest.GetUpdatedAt().After(review.LastDetailFetch), nil
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
				var reviews []*github.PullRequestReview

				log.Debug("Fetching page of pull request reviews", "repo", rf.repo.String(), "pr", pr.GetNumber(), "page", pg)
				// https://docs.github.com/en/rest/pulls/reviews#list-reviews-for-a-pull-request
				reviews, res, err = cfg.Client.PullRequests.ListReviews(cx, rf.repo.GetOwner(), rf.repo.GetName(), pr.GetNumber(), &github.ListOptions{
					Page:    pg,
					PerPage: DEFAULT_PER_PAGE,
				})
				if err != nil {
					return
				}
				for _, ghReview := range reviews {
					var review *PullRequestReview
					review, err = LoadPullRequestReview(rf.rootPath, rf.repo, pr.GetNumber(), ghReview.GetID(), false)
					if err != nil {
						return
					}
					review.Review = ghReview
					review.PullRequestNumber = pr.GetNumber()
					review.LastDetailFetch = time.Now()
					if err = review.Save(rf.rootPath, rf.repo, cfg.PrettyJSON); err != nil {
						return
					}
				}
				done = len(reviews) < DEFAULT_PER_PAGE
				return
			})
		if err != nil {
			return nil, err
		}

		pr.LastReviewsFetch = time.Now()
		if err := pr.Save(rf.rootPath, rf.repo, cfg.PrettyJSON); err != nil {
			return nil, err
		}
	}
	return rf.makeReviewCommentsFetcher(log)
}

func (rf *pullRequestReviewsFetcher) makeReviewCommentsFetcher(log Logger) ([]fetcher, error) {
	// A list of the pull request reviews for which we should be fetching
	// comments.
	fetchReviewComments := []*PullRequestReview{}
	// First we need to run through all pull requests, so we know when they were
	// last updated.
	prsPath := repoPullRequestsPath(rf.rootPath, rf.repo.GetOwner(), rf.repo.GetName())
	pattern := filepath.Join(prsPath, "*", DETAIL_FILENAME)
	prFiles, err := filepath.Glob(pattern)
	if err != nil {
		return nil, fmt.Errorf("unable to scan for pull request details files with pattern %s: %v", pattern, err)
	}

	for _, prf := range prFiles {
		pr, err := LoadPullRequestDirect(prf, true)
		if err != nil {
			return nil, err
		}
		// Now we scan all pull request reviews for this pull request
		prReviewsPath := pullRequestReviewsPath(rf.rootPath, rf.repo.GetOwner(), rf.repo.GetName(), pr.GetNumber())
		pat := filepath.Join(prReviewsPath, "*", DETAIL_FILENAME)
		reviewFiles, err := filepath.Glob(pat)
		if err != nil {
			return nil, fmt.Errorf("unable to scan for pull request review detail files with pattern %s: %v", pat, err)
		}
		for _, fn := range reviewFiles {
			review, err := LoadPullRequestReviewDirect(fn, true)
			if err != nil {
				return nil, err
			}
			if pr.PullRequest.GetUpdatedAt().After(review.LastCommentsFetch) {
				fetchReviewComments = append(fetchReviewComments, review)
			}
		}
	}

	fetchers := []fetcher{
		newPullRequestReviewCommentsFetcher(rf.rootPath, rf.repo, fetchReviewComments),
	}
	return fetchers, nil
}
