package ghere

import (
	"context"

	"github.com/google/go-github/v48/github"
)

type PullRequestReview struct {
	Review *github.PullRequestReview `json:"review"`
}

type pullRequestReviewFetcher struct {
	pullsPath string
	repo      *Repository
	prNums    []int
}

func newPullRequestReviewFetcher(pullsPath string, repo *Repository, prNums []int) *pullRequestReviewFetcher {
	return &pullRequestReviewFetcher{
		pullsPath: pullsPath,
		repo:      repo,
		prNums:    prNums,
	}
}

func (rf *pullRequestReviewFetcher) fetch(ctx context.Context, cfg *FetchConfig, log Logger) ([]fetcher, error) {
	return nil, nil
}
