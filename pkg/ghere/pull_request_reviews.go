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

	LastDetailFetch   time.Time `json:"last_detail_fetch"`
	LastCommentsFetch time.Time `json:"last_comments_fetch"`
}

func (r *PullRequestReview) GetPath(prReviewsPath string) string {
	return filepath.Join(prReviewsPath, fmt.Sprintf("%d", r.Review.GetID()))
}

func (r *PullRequestReview) GetDetailPath(prReviewsPath string) string {
	return filepath.Join(r.GetPath(prReviewsPath), DETAIL_FILENAME)
}

// pullRequestReviewsFetcher fetches reviews for a number of PRs.
type pullRequestReviewsFetcher struct {
	prsPath      string
	repo         *Repository
	pullRequests []*PullRequest
}

func newPullRequestReviewsFetcher(prsPath string, repo *Repository, pullRequests []*PullRequest) *pullRequestReviewsFetcher {
	return &pullRequestReviewsFetcher{
		prsPath:      prsPath,
		repo:         repo,
		pullRequests: pullRequests,
	}
}

func (rf *pullRequestReviewsFetcher) fetch(ctx context.Context, cfg *FetchConfig, log Logger) ([]fetcher, error) {
	for _, pr := range rf.pullRequests {
		log.Info("Fetching pull request reviews", "pr", pr.GetNumber())
		prReviewsPath := pr.GetReviewsPath(rf.prsPath)
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

		err = rateLimitedPaginated(startPage, log, func(pg int) (res *github.Response, done bool, err error) {
			var reviews []*github.PullRequestReview

			log.Info("Fetching page of pull request reviews", "pr", pr.GetNumber(), "page", pg)
			// https://docs.github.com/en/rest/pulls/reviews#list-reviews-for-a-pull-request
			reviews, res, err = cfg.Client.PullRequests.ListReviews(ctx, rf.repo.GetOwner(), rf.repo.GetName(), pr.GetNumber(), &github.ListOptions{
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
				revPath := rev.GetDetailPath(prReviewsPath)
				if err = readJSONFileOrEmpty(revPath, rev); err != nil {
					return
				}
				rev.Review = review
				rev.LastDetailFetch = time.Now()
				if err = writeJSONFile(revPath, rev, cfg.PrettyJSON); err != nil {
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
	}
	return rf.makeReviewCommentsFetcher(log)
}

func (rf *pullRequestReviewsFetcher) makeReviewCommentsFetcher(log Logger) ([]fetcher, error) {
	return nil, nil
}
