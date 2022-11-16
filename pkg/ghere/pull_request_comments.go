package ghere

import (
	"context"

	"github.com/google/go-github/v48/github"
)

type PullRequestComment struct {
	Comment *github.PullRequestComment `json:"comment"`
}

type pullRequestCommentsFetcher struct {
	pullsPath    string
	repo         *Repository
	pullRequests []*PullRequest
}

func newPullRequestCommentsFetcher(pullsPath string, repo *Repository, pullRequests []*PullRequest) *pullRequestCommentsFetcher {
	return &pullRequestCommentsFetcher{
		pullsPath:    pullsPath,
		repo:         repo,
		pullRequests: pullRequests,
	}
}

func (cf *pullRequestCommentsFetcher) fetch(ctx context.Context, cfg *FetchConfig, log Logger) ([]fetcher, error) {
	return nil, nil
}
