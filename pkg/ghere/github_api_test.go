package ghere_test

import (
	"context"
	"fmt"

	"github.com/google/go-github/v48/github"
	"github.com/informalsystems/ghere/pkg/ghere"
)

type MockGitHubClient struct {
	Repositories              map[string]*github.Repository
	Labels                    map[string][]*github.Label
	PullRequests              map[string][]*github.PullRequest
	PullRequestReviews        map[string]map[int][]*github.PullRequestReview
	PullRequestReviewComments map[string]map[int]map[int64][]*github.PullRequestComment
	PullRequestComments       map[string]map[int][]*github.PullRequestComment
	Issues                    map[string][]*github.Issue
	IssueComments             map[string]map[int][]*github.IssueComment
}

var _ ghere.GitHubClient = (*MockGitHubClient)(nil)

// GetRepository implements ghere.GitHubClient
func (c *MockGitHubClient) GetRepository(ctx context.Context, owner string, name string) (*github.Repository, error) {
	return getForRepo(c.Repositories, owner, name)
}

// ListIssueComments implements ghere.GitHubClient
func (c *MockGitHubClient) ListIssueComments(ctx context.Context, owner string, name string, issueNum int, page int) ([]*github.IssueComment, bool, error) {
	return getPageForIssueOrPR(c.IssueComments, owner, name, issueNum, page, "issue")
}

// ListPullRequestComments implements ghere.GitHubClient
func (c *MockGitHubClient) ListPullRequestComments(ctx context.Context, owner string, name string, prNum int, page int) ([]*github.PullRequestComment, bool, error) {
	return getPageForIssueOrPR(c.PullRequestComments, owner, name, prNum, page, "pull request")
}

// ListPullRequestReviewComments implements ghere.GitHubClient
func (c *MockGitHubClient) ListPullRequestReviewComments(ctx context.Context, owner string, name string, prNum int, reviewID int64, page int) ([]*github.PullRequestComment, bool, error) {
	reviewComments, err := getForIssueOrPR(c.PullRequestReviewComments, owner, name, prNum, "pull request")
	if err != nil {
		return nil, false, err
	}
	comments, exists := reviewComments[reviewID]
	if !exists {
		return nil, false, fmt.Errorf("no such pull request review %d for PR %d of %s/%s", reviewID, prNum, owner, name)
	}
	return getListPage(comments, page)
}

// ListPullRequestReviews implements ghere.GitHubClient
func (c *MockGitHubClient) ListPullRequestReviews(ctx context.Context, owner string, name string, prNum int, page int) ([]*github.PullRequestReview, bool, error) {
	return getPageForIssueOrPR(c.PullRequestReviews, owner, name, prNum, page, "pull request")
}

// ListRepositoryIssues implements ghere.GitHubClient
func (c *MockGitHubClient) ListRepositoryIssues(ctx context.Context, owner string, name string, page int) ([]*github.Issue, bool, error) {
	return getPageForRepo(c.Issues, owner, name, page)
}

// ListRepositoryLabels implements ghere.GitHubClient
func (c *MockGitHubClient) ListRepositoryLabels(ctx context.Context, owner string, name string, page int) ([]*github.Label, bool, error) {
	return getPageForRepo(c.Labels, owner, name, page)
}

// ListRepositoryPullRequests implements ghere.GitHubClient
func (c *MockGitHubClient) ListRepositoryPullRequests(ctx context.Context, owner string, name string, page int) ([]*github.PullRequest, bool, error) {
	return getPageForRepo(c.PullRequests, owner, name, page)
}

func getPageForIssueOrPR[V any](m map[string]map[int][]V, owner, name string, n, page int, tp string) ([]V, bool, error) {
	var empty []V
	allItems, err := getForIssueOrPR(m, owner, name, n, tp)
	if err != nil {
		return empty, false, err
	}
	return getListPage(allItems, page)
}

func getForIssueOrPR[V any](m map[string]map[int]V, owner, name string, n int, tp string) (V, error) {
	var empty V
	forRepo, err := getForRepo(m, owner, name)
	if err != nil {
		return empty, err
	}
	forIssueOrPR, exists := forRepo[n]
	if !exists {
		return empty, fmt.Errorf("no such %s %d for %s/%s", tp, n, owner, name)
	}
	return forIssueOrPR, nil
}

func getPageForRepo[V any](m map[string][]V, owner, name string, page int) ([]V, bool, error) {
	var empty []V
	allItems, err := getForRepo(m, owner, name)
	if err != nil {
		return empty, false, err
	}
	return getListPage(allItems, page)
}

func getForRepo[V any](m map[string]V, owner, name string) (V, error) {
	var empty V
	id := owner + "/" + name
	v, exists := m[id]
	if !exists {
		return empty, fmt.Errorf("no such repository: %s", id)
	}
	return v, nil
}

func getListPage[V any](l []V, page int) ([]V, bool, error) {
	startIdx := (page - 1) * ghere.DEFAULT_PER_PAGE
	if startIdx >= len(l) {
		return []V{}, true, nil
	}
	endIdx := page * ghere.DEFAULT_PER_PAGE
	if startIdx+endIdx >= len(l) {
		endIdx = len(l)
	}
	items := l[startIdx:endIdx]
	return items, len(items) < ghere.DEFAULT_PER_PAGE, nil
}
