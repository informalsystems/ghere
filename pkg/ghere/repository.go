package ghere

import (
	"context"
	"time"

	"github.com/google/go-github/v48/github"
)

type Repository struct {
	Repository *github.Repository `json:"repository"`

	LastDetailFetch       time.Time `json:"last_detail_fetch"`
	LastPullRequestsFetch time.Time `json:"last_pull_requests_fetch"`
	LastIssuesFetch       time.Time `json:"last_issues_fetch"`
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
		repo:     &Repository{},
	}
}

func (rf *repoFetcher) fetch(ctx context.Context, cfg *FetchConfig, log Logger) ([]fetcher, error) {
	detailFile := repoDetailPath(rf.rootPath, rf.owner, rf.name)
	if err := readJSONFileOrEmpty(detailFile, rf.repo); err != nil {
		return nil, err
	}
	err := rateLimited(log, func() (res *github.Response, err error) {
		rf.repo.Repository, res, err = cfg.Client.Repositories.Get(ctx, rf.owner, rf.name)
		return
	})
	if err != nil {
		return nil, err
	}
	rf.repo.LastDetailFetch = time.Now()
	if err := writeJSONFile(detailFile, rf.repo, cfg.PrettyJSON); err != nil {
		return nil, err
	}
	fetchers := []fetcher{newCodeFetcher(rf.rootPath, rf.repo)}
	if rf.repo.Repository.GetUpdatedAt().After(rf.repo.LastPullRequestsFetch) {
		fetchers = append(fetchers, newPullRequestsFetcher(
			rf.rootPath,
			rf.repo,
		))
	}
	return fetchers, nil
}
