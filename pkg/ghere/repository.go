package ghere

import (
	"context"
	"path/filepath"
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

func (r *Repository) GetPath(rootPath string) string {
	return filepath.Join(rootPath, r.GetOwner(), r.GetName())
}

func (r *Repository) GetDetailPath(rootPath string) string {
	return filepath.Join(r.GetPath(rootPath), DETAIL_FILENAME)
}

func (r *Repository) GetPullRequestsPath(rootPath string) string {
	return filepath.Join(r.GetPath(rootPath), "pull-requests")
}

func (r *Repository) GetCodePath(rootPath string) string {
	return filepath.Join(r.GetPath(rootPath), "code")
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
		// Fill out the bare minimum information for the GetOwner and GetName
		// methods to work.
		repo: &Repository{
			Repository: &github.Repository{
				Owner: &github.User{
					Login: &owner,
				},
				Name: &name,
			},
		},
	}
}

func (rf *repoFetcher) fetch(ctx context.Context, cfg *FetchConfig, log Logger) ([]fetcher, error) {
	repoFile := rf.repo.GetDetailPath(rf.rootPath)
	if err := readJSONFileOrEmpty(repoFile, rf.repo); err != nil {
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
	if err := writeJSONFile(repoFile, rf.repo, cfg.PrettyJSON); err != nil {
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
