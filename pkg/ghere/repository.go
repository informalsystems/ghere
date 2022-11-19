package ghere

import (
	"context"
	"fmt"
	"time"

	"github.com/google/go-github/v48/github"
)

type Repository struct {
	Repository *github.Repository `json:"repository"`

	LastDetailFetch              time.Time `json:"last_detail_fetch"`
	LastPullRequestsFetch        time.Time `json:"last_pull_requests_fetch"`
	LastPullRequestReviewsFetch  time.Time `json:"last_pull_request_reviews_fetch"`
	LastPullRequestCommentsFetch time.Time `json:"last_pull_request_comments_fetch"`
	LastIssuesFetch              time.Time `json:"last_issues_fetch"`
	LastIssueCommentsFetch       time.Time `json:"last_issue_comments_fetch"`
	LastLabelsFetch              time.Time `json:"last_labels_fetch"`
}

func LoadRepository(rootPath, owner, name string, mustExist bool) (*Repository, error) {
	var err error
	repo := &Repository{}
	path := repoDetailPath(rootPath, owner, name)
	if mustExist {
		err = readJSONFile(path, repo)
	} else {
		err = readJSONFileOrEmpty(path, repo)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to read repository detail file: %v", err)
	}
	return repo, nil
}

func (r *Repository) MustFetchPullRequests() bool {
	return r.Repository.GetUpdatedAt().After(r.LastPullRequestsFetch) || r.MustFetchPullRequestReviews() || r.MustFetchPullRequestComments()
}

func (r *Repository) MustFetchPullRequestReviews() bool {
	return r.Repository.GetUpdatedAt().After(r.LastPullRequestReviewsFetch)
}

func (r *Repository) MustFetchPullRequestComments() bool {
	return r.Repository.GetUpdatedAt().After(r.LastPullRequestCommentsFetch)
}

func (r *Repository) MustFetchIssues() bool {
	return r.Repository.GetUpdatedAt().After(r.LastIssuesFetch) || r.MustFetchIssueComments()
}

func (r *Repository) MustFetchIssueComments() bool {
	return r.Repository.GetUpdatedAt().After(r.LastIssueCommentsFetch)
}

func (r *Repository) MustFetchLabels() bool {
	return r.Repository.GetUpdatedAt().After(r.LastLabelsFetch)
}

func (r *Repository) String() string {
	return r.GetOwner() + "/" + r.GetName()
}

func (r *Repository) GetOwner() string {
	return r.Repository.GetOwner().GetLogin()
}

func (r *Repository) GetName() string {
	return r.Repository.GetName()
}

func (r *Repository) Save(rootPath string, prettyJSON bool) error {
	path := repoDetailPath(rootPath, r.GetOwner(), r.GetName())
	if err := writeJSONFile(path, r, prettyJSON); err != nil {
		return fmt.Errorf("failed to write repository detail file: %v", err)
	}
	return nil
}

type repoFetcher struct {
	rootPath string
	owner    string
	name     string
	repo     *Repository
}

var _ fetcher = (*repoFetcher)(nil)

func newRepoFetcher(rootPath, owner, name string) *repoFetcher {
	return &repoFetcher{
		rootPath: rootPath,
		owner:    owner,
		name:     name,
	}
}

func (rf *repoFetcher) fetch(ctx context.Context, cfg *FetchConfig, log Logger) ([]fetcher, error) {
	var err error
	rf.repo, err = LoadRepository(rf.rootPath, rf.owner, rf.name, false)
	if err != nil {
		return nil, err
	}
	rf.repo.Repository, err = cfg.Client.GetRepository(ctx, rf.owner, rf.name)
	if err != nil {
		return nil, err
	}
	rf.repo.LastDetailFetch = time.Now()
	if err := rf.repo.Save(rf.rootPath, cfg.PrettyJSON); err != nil {
		return nil, err
	}
	fetchers := []fetcher{newCodeFetcher(rf.rootPath, rf.repo)}
	if rf.repo.MustFetchLabels() {
		fetchers = append(fetchers, newLabelsFetcher(
			rf.rootPath,
			rf.repo,
		))
	}
	if rf.repo.MustFetchPullRequests() {
		fetchers = append(fetchers, newPullRequestsFetcher(
			rf.rootPath,
			rf.repo,
		))
	}
	if rf.repo.MustFetchIssues() {
		fetchers = append(fetchers, newIssuesFetcher(
			rf.rootPath,
			rf.repo,
		))
	}
	return fetchers, nil
}
