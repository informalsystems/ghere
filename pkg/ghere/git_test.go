package ghere_test

import (
	"context"

	"github.com/google/go-github/v48/github"
	"github.com/informalsystems/ghere/pkg/ghere"
)

// MockGitHubCredentialProvider does nothing.
type MockGitHubCredentialProvider struct{}

var _ ghere.GitHubCredentialProvider = (*MockGitHubCredentialProvider)(nil)

// GetGitHubCredentials implements ghere.GitHubCredentialProvider
func (*MockGitHubCredentialProvider) GetGitHubCredentials(ctx context.Context) (*ghere.GitHubCredentials, error) {
	return nil, nil
}

// MockGitHubRepositoryUpdater does nothing.
type MockGitHubRepositoryUpdater struct{}

var _ ghere.GitHubRepositoryUpdater = (*MockGitHubRepositoryUpdater)(nil)

// CloneOrUpdateRepository implements ghere.GitHubRepositoryUpdater
func (*MockGitHubRepositoryUpdater) CloneOrUpdateRepository(ctx context.Context, repoDir string, repo *github.Repository, credentialProvider ghere.GitHubCredentialProvider, log ghere.Logger) error {
	return nil
}
