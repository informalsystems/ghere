package ghere

import (
	"context"
	"fmt"
	"time"

	"github.com/google/go-github/v48/github"
)

// GitHubClient provides an interface through which we can access GitHub data.
// This allows us to mock out the client during testing, and allows users of the
// API to change the behaviour of the client if they so wish.
//
// The [NewGitHubClient] method is provided by default.
type GitHubClient interface {
	GetRepository(ctx context.Context, owner, name string) (*github.Repository, error)
	ListRepositoryLabels(ctx context.Context, owner, name string, page int) ([]*github.Label, bool, error)
	ListRepositoryPullRequests(ctx context.Context, owner, name string, page int) ([]*github.PullRequest, bool, error)
	ListPullRequestReviews(ctx context.Context, owner, name string, prNum int, page int) ([]*github.PullRequestReview, bool, error)
	ListPullRequestReviewComments(ctx context.Context, owner, name string, prNum int, reviewID int64, page int) ([]*github.PullRequestComment, bool, error)
	ListPullRequestComments(ctx context.Context, owner, name string, prNum int, page int) ([]*github.PullRequestComment, bool, error)
	ListRepositoryIssues(ctx context.Context, owner, name string, page int) ([]*github.Issue, bool, error)
	ListIssueComments(ctx context.Context, owner, name string, issueNum int, page int) ([]*github.IssueComment, bool, error)
}

type githubClient struct {
	client  *github.Client
	retries int
	timeout time.Duration
	log     Logger
}

var _ GitHubClient = (*githubClient)(nil)

// NewGitHubClient constructs a [GitHubClient] implementation that automatically
// handles rate limiting (by waiting until the rate limit reset time when the
// rate limit is hit) as well as request timeouts and retries.
func NewGitHubClient(client *github.Client, retries int, timeout time.Duration, log Logger) GitHubClient {
	return &githubClient{
		client:  client,
		retries: retries,
		timeout: timeout,
		log:     log,
	}
}

func (c *githubClient) GetRepository(ctx context.Context, owner, name string) (*github.Repository, error) {
	var repo *github.Repository
	c.log.Info("Get repository", "repo", owner+"/"+name)
	err := c.callRateLimited(ctx, func(cx context.Context) (res *github.Response, err error) {
		repo, res, err = c.client.Repositories.Get(cx, owner, name)
		return
	})
	if err != nil {
		return nil, err
	}
	return repo, nil
}

func (c *githubClient) ListRepositoryLabels(ctx context.Context, owner, name string, page int) ([]*github.Label, bool, error) {
	var labels []*github.Label
	c.log.Info("List repository labels", "repo", owner+"/"+name, "page", page)
	err := c.callRateLimited(ctx, func(cx context.Context) (res *github.Response, err error) {
		labels, res, err = c.client.Issues.ListLabels(cx, owner, name, &github.ListOptions{
			Page:    page,
			PerPage: DEFAULT_PER_PAGE,
		})
		return
	})
	if err != nil {
		return nil, false, err
	}
	return labels, len(labels) < DEFAULT_PER_PAGE, nil
}

func (c *githubClient) ListRepositoryPullRequests(ctx context.Context, owner, name string, page int) ([]*github.PullRequest, bool, error) {
	var prs []*github.PullRequest
	c.log.Info("List repository pull requests", "repo", owner+"/"+name, "page", page)
	err := c.callRateLimited(ctx, func(cx context.Context) (res *github.Response, err error) {
		prs, res, err = c.client.PullRequests.List(cx, owner, name, &github.PullRequestListOptions{
			State:     "all",
			Sort:      "created",
			Direction: "asc",
			ListOptions: github.ListOptions{
				Page:    page,
				PerPage: DEFAULT_PER_PAGE,
			},
		})
		return
	})
	if err != nil {
		return nil, false, err
	}
	return prs, len(prs) < DEFAULT_PER_PAGE, nil
}

func (c *githubClient) ListPullRequestReviews(ctx context.Context, owner, name string, prNum int, page int) ([]*github.PullRequestReview, bool, error) {
	var reviews []*github.PullRequestReview
	c.log.Info("List repository pull request reviews", "repo", owner+"/"+name, "pr", prNum, "page", page)
	err := c.callRateLimited(ctx, func(cx context.Context) (res *github.Response, err error) {
		reviews, res, err = c.client.PullRequests.ListReviews(cx, owner, name, prNum, &github.ListOptions{
			Page:    page,
			PerPage: DEFAULT_PER_PAGE,
		})
		return
	})
	if err != nil {
		return nil, false, err
	}
	return reviews, len(reviews) < DEFAULT_PER_PAGE, nil
}

func (c *githubClient) ListPullRequestReviewComments(ctx context.Context, owner, name string, prNum int, reviewID int64, page int) ([]*github.PullRequestComment, bool, error) {
	var comments []*github.PullRequestComment
	c.log.Info("List pull request review comments", "repo", owner+"/"+name, "pr", prNum, "reviewID", reviewID, "page", page)
	err := c.callRateLimited(ctx, func(cx context.Context) (res *github.Response, err error) {
		comments, res, err = c.client.PullRequests.ListReviewComments(
			cx,
			owner,
			name,
			prNum,
			reviewID,
			&github.ListOptions{
				Page:    page,
				PerPage: DEFAULT_PER_PAGE,
			},
		)
		return
	})
	if err != nil {
		return nil, false, err
	}
	return comments, len(comments) < DEFAULT_PER_PAGE, nil
}

func (c *githubClient) ListPullRequestComments(ctx context.Context, owner, name string, prNum int, page int) ([]*github.PullRequestComment, bool, error) {
	var comments []*github.PullRequestComment
	c.log.Info("List pull request comments", "repo", owner+"/"+name, "pr", prNum, "page", page)
	err := c.callRateLimited(ctx, func(cx context.Context) (res *github.Response, err error) {
		comments, res, err = c.client.PullRequests.ListComments(
			cx,
			owner,
			name,
			prNum,
			&github.PullRequestListCommentsOptions{
				Sort:      "created",
				Direction: "asc",
				ListOptions: github.ListOptions{
					Page:    page,
					PerPage: DEFAULT_PER_PAGE,
				},
			},
		)
		return
	})
	if err != nil {
		return nil, false, err
	}
	return comments, len(comments) < DEFAULT_PER_PAGE, nil
}

func (c *githubClient) ListRepositoryIssues(ctx context.Context, owner, name string, page int) ([]*github.Issue, bool, error) {
	var issues []*github.Issue
	c.log.Info("List repository issues", "repo", owner+"/"+name, "page", page)
	err := c.callRateLimited(ctx, func(cx context.Context) (res *github.Response, err error) {
		issues, res, err = c.client.Issues.ListByRepo(cx, owner, name, &github.IssueListByRepoOptions{
			State:     "all",
			Sort:      "created",
			Direction: "asc",
			ListOptions: github.ListOptions{
				Page:    page,
				PerPage: DEFAULT_PER_PAGE,
			},
		},
		)
		return
	})
	if err != nil {
		return nil, false, err
	}
	return issues, len(issues) < DEFAULT_PER_PAGE, nil
}

func (c *githubClient) ListIssueComments(ctx context.Context, owner, name string, issueNum int, page int) ([]*github.IssueComment, bool, error) {
	var comments []*github.IssueComment
	c.log.Info("List issue comments", "repo", owner+"/"+name, "issue", issueNum, "page", page)
	err := c.callRateLimited(ctx, func(cx context.Context) (res *github.Response, err error) {
		// For some reason the issue comment listing Go API requires pointers to
		// strings as parameters instead of raw strings
		sortParam := "created"
		dirParam := "asc"
		comments, res, err = c.client.Issues.ListComments(cx, owner, name, issueNum, &github.IssueListCommentsOptions{
			Sort:      &sortParam,
			Direction: &dirParam,
			ListOptions: github.ListOptions{
				Page:    page,
				PerPage: DEFAULT_PER_PAGE,
			},
		},
		)
		return
	})
	if err != nil {
		return nil, false, err
	}
	return comments, len(comments) < DEFAULT_PER_PAGE, nil
}

func (c *githubClient) callRateLimited(ctx context.Context, fn func(cx context.Context) (*github.Response, error)) error {
	var res *github.Response
	var err error

	for {
		res, err = c.retryWithTimeout(ctx, fn)
		if err == nil {
			break
		}
		if res.Rate.Remaining > 0 {
			return err
		}
		c.log.Warn("GitHub rate limit hit, waiting until reset time", "limit", res.Rate.Limit, "reset", res.Rate.Reset.Local().String())
		time.Sleep(time.Until(res.Rate.Reset.Time) + time.Second)
	}
	c.log.Debug("Rate limiting", "limit", res.Rate.Limit, "remaining", res.Rate.Remaining)
	return nil
}

type retryResult struct {
	res *github.Response
	err error
}

func (c *githubClient) retryWithTimeout(ctx context.Context, fn func(cx context.Context) (*github.Response, error)) (*github.Response, error) {
	for attempt := 0; attempt < c.retries; attempt++ {
		attemptCtx, cancelAttempt := context.WithCancel(ctx)
		resChan := make(chan retryResult)
		go func() {
			defer cancelAttempt()
			res, err := fn(attemptCtx)
			resChan <- retryResult{res, err}
		}()
		select {
		case res := <-resChan:
			return res.res, res.err
		case <-time.After(c.timeout):
			c.log.Warn("Timed out while attempting GitHub request; retrying", "timeout", c.timeout.String(), "attempt", attempt+1, "retries", c.retries)
			cancelAttempt()
		}
	}
	return nil, fmt.Errorf("failed to execute GitHub request %d times", c.retries)
}
