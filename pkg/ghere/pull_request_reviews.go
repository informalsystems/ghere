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
			review := &PullRequestReview{}
			if err := readJSONFile(fn, review); err != nil {
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
				for _, review := range reviews {
					rev := &PullRequestReview{
						Review: review,
					}
					revPath := pullRequestReviewDetailPath(rf.rootPath, rf.repo.GetOwner(), rf.repo.GetName(), pr.GetNumber(), review.GetID())
					if err = readJSONFileOrEmpty(revPath, rev); err != nil {
						err = fmt.Errorf("failed to read pull request review detail file %s: %v", revPath, err)
						return
					}
					rev.Review = review
					rev.PullRequestNumber = pr.GetNumber()
					rev.LastDetailFetch = time.Now()
					if err = writeJSONFile(revPath, rev, cfg.PrettyJSON); err != nil {
						err = fmt.Errorf("failed to write pull request review detail file %s: %v", revPath, err)
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
		prDetailPath := pullRequestDetailPath(rf.rootPath, rf.repo.GetOwner(), rf.repo.GetName(), pr.GetNumber())
		if err := writeJSONFile(prDetailPath, pr, cfg.PrettyJSON); err != nil {
			return nil, fmt.Errorf("failed to update last reviews fetch time for pull request %s: %v", prDetailPath, err)
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
		pr := &PullRequest{}
		if err := readJSONFile(prf, pr); err != nil {
			return nil, fmt.Errorf("failed to read pull request detail file %s: %v", prf, err)
		}
		// Now we scan all pull request reviews for this pull request
		pat := pullRequestReviewsPath(rf.rootPath, rf.repo.GetOwner(), rf.repo.GetName(), pr.GetNumber())
		reviewFiles, err := filepath.Glob(pattern)
		if err != nil {
			return nil, fmt.Errorf("unable to scan for pull request review detail files with pattern %s: %v", pat, err)
		}
		for _, fn := range reviewFiles {
			review := &PullRequestReview{}
			if err := readJSONFile(fn, review); err != nil {
				return nil, fmt.Errorf("failed to read pull request review detail file %s: %v", fn, err)
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
