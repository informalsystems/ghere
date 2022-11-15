package ghere

import (
	"context"

	"github.com/google/go-github/v48/github"
)

type PullRequestComment struct {
	Comment *github.PullRequestComment `json:"comment"`
}

type pullRequestCommentsFetcher struct {
	pullsPath string
	repo      *Repository
	prNums    []int
}

func newPullRequestCommentsFetcher(pullsPath string, repo *Repository, prNums []int) *pullRequestCommentsFetcher {
	return &pullRequestCommentsFetcher{
		pullsPath: pullsPath,
		repo:      repo,
		prNums:    prNums,
	}
}

func (cf *pullRequestCommentsFetcher) fetch(ctx context.Context, cfg *FetchConfig, log Logger) ([]fetcher, error) {
	return nil, nil
}
