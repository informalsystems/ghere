package ghere

import (
	"context"
	"fmt"
)

type codeFetcher struct {
	codePath string
	repo     *Repository
}

var _ fetcher = (*codeFetcher)(nil)

func newCodeFetcher(codePath string, repo *Repository) *codeFetcher {
	return &codeFetcher{
		codePath: codePath,
		repo:     repo,
	}
}

func (cf *codeFetcher) fetch(ctx context.Context, cfg *FetchConfig, log Logger) ([]fetcher, error) {
	repo := cf.repo.Repository
	if len(repo.GetSSHURL()) == 0 {
		return nil, fmt.Errorf("repository %s is missing its SSH URL", repo)
	}
	log.Info("Cloning/updating code repository from GitHub", "repo", repo.GetSSHURL(), "dest", cf.codePath)
	if err := cloneOrUpdateRepository(ctx, cf.codePath, repo.GetSSHURL(), cfg.SSHPrivKeyFile, cfg.SSHPrivKeyFilePassword); err != nil {
		return nil, err
	}
	log.Info("Successfully fetched latest code from GitHub", "repo", repo.GetSSHURL(), "dest", cf.codePath)
	return nil, nil
}
