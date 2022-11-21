package ghere

import (
	"context"
	"fmt"
)

type codeFetcher struct {
	rootPath string
	repo     *Repository
}

var _ fetcher = (*codeFetcher)(nil)

func newCodeFetcher(rootPath string, repo *Repository) *codeFetcher {
	return &codeFetcher{
		rootPath: rootPath,
		repo:     repo,
	}
}

func (cf *codeFetcher) fetch(ctx context.Context, cfg *FetchConfig, log Logger) ([]fetcher, error) {
	repo := cf.repo.Repository
	if len(repo.GetSSHURL()) == 0 {
		return nil, fmt.Errorf("repository %s is missing its SSH URL", repo)
	}
	codePath := repoCodePath(cf.rootPath, cf.repo.GetOwner(), cf.repo.GetName())
	if err := cfg.RepoUpdater.CloneOrUpdateRepository(ctx, codePath, repo, cfg.CredentialProvider, log); err != nil {
		return nil, err
	}
	return nil, nil
}
