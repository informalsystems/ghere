package ghere

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/transport"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/go-git/go-git/v5/plumbing/transport/ssh"
	"github.com/google/go-github/v48/github"
)

const (
	GITHUB_PASSWORD_ENVVAR      string = "GITHUB_PASSWORD"
	SSH_PRIVKEY_PASSWORD_ENVVAR string = "SSH_PRIVKEY_PASSWORD"
)

type GitHubCredentials struct {
	PubKeys   *ssh.PublicKeys
	BasicAuth *http.BasicAuth
}

// GitHubCredentialProvider provides a way to access GitHub credentials in
// order to fetch Git repositories from protected remote repositories.
type GitHubCredentialProvider interface {
	GetGitHubCredentials(ctx context.Context) (*GitHubCredentials, error)
}

type envVarCredentialProvider struct {
	privKeyFile string
	username    string
}

var _ GitHubCredentialProvider = (*envVarCredentialProvider)(nil)

// NewGitHubEnvVarCredentialProvider creates a [GitHubCredentialProvider] that
// gets passwords/secrets from environment variables.
func NewGitHubEnvVarCredentialProvider(privKeyFile, username string) GitHubCredentialProvider {
	return &envVarCredentialProvider{
		privKeyFile: privKeyFile,
		username:    username,
	}
}

func (cp *envVarCredentialProvider) GetGitHubCredentials(ctx context.Context) (*GitHubCredentials, error) {
	select {
	case <-ctx.Done():
		return nil, errors.New("credentials fetch cancelled")
	default:
	}
	pubKeys, err := cp.getSSHKeys()
	if err != nil {
		return nil, fmt.Errorf("failed to load SSH key: %v", err)
	}
	basicAuth := cp.getBasicAuth()
	return &GitHubCredentials{
		PubKeys:   pubKeys,
		BasicAuth: basicAuth,
	}, nil
}

func (cp *envVarCredentialProvider) getBasicAuth() *http.BasicAuth {
	return &http.BasicAuth{
		Username: cp.username,
		Password: os.Getenv(GITHUB_PASSWORD_ENVVAR),
	}
}

func (cp *envVarCredentialProvider) getSSHKeys() (*ssh.PublicKeys, error) {
	privKeyFilePassword := os.Getenv(SSH_PRIVKEY_PASSWORD_ENVVAR)
	exists, err := fileExists(cp.privKeyFile)
	if err != nil || !exists {
		return nil, nil
	}
	pubKeys, err := ssh.NewPublicKeysFromFile("git", cp.privKeyFile, privKeyFilePassword)
	if err != nil {
		return nil, fmt.Errorf("unable to generate Git public keys from %s: %v", cp.privKeyFile, err)
	}
	return pubKeys, nil
}

type GitHubRepositoryUpdater interface {
	CloneOrUpdateRepository(ctx context.Context, repoDir string, repo *github.Repository, credentialProvider GitHubCredentialProvider, log Logger) error
}

type githubRepositoryUpdater struct{}

var _ GitHubRepositoryUpdater = (*githubRepositoryUpdater)(nil)

func NewGitHubRepositoryUpdater() GitHubRepositoryUpdater {
	return &githubRepositoryUpdater{}
}

type githubAuthMethod struct {
	repoURL string
	auth    transport.AuthMethod
}

func (u *githubRepositoryUpdater) CloneOrUpdateRepository(ctx context.Context, repoDir string, repo *github.Repository, credentialProvider GitHubCredentialProvider, log Logger) error {
	creds, err := credentialProvider.GetGitHubCredentials(ctx)
	if err != nil {
		return err
	}
	repoID := repo.GetOwner().GetLogin() + "/" + repo.GetName()
	authMethods := make([]*githubAuthMethod, 0)
	if len(repo.GetSSHURL()) > 0 && creds.PubKeys != nil {
		log.Debug("Configured SSH credentials", "repo", repoID)
		authMethods = append(authMethods, &githubAuthMethod{
			repoURL: repo.GetSSHURL(),
			auth:    creds.PubKeys,
		})
	}
	if len(repo.GetCloneURL()) > 0 && creds.BasicAuth != nil {
		log.Debug("Configured HTTP credentials", "repo", repo.GetOwner().GetLogin()+"/"+repo.GetName())
		authMethods = append(authMethods, &githubAuthMethod{
			repoURL: repo.GetCloneURL(),
			auth:    creds.BasicAuth,
		})
	}
	if len(authMethods) == 0 {
		log.Warn("No SSH or HTTP(S) credentials specified for repository", "repo", repoID)
	}
	gitDir := filepath.Join(repoDir, ".git")
	exists, err := dirExists(gitDir)
	if err != nil {
		return fmt.Errorf("failed to access Git repository directory %s: %v", gitDir, err)
	}
	for _, method := range authMethods {
		if exists {
			log.Info("Attempting to pull latest changes from repository", "repoDir", repoDir, "repoURL", method.repoURL)
			err = updateRepository(ctx, repoDir, method.auth)
			if err == nil {
				log.Info("Successfully pulled latest changes from repository", "repoDir", repoDir, "repoURL", method.repoURL)
				return nil
			}
			log.Warn("Failed to update repository", "repoDir", repoDir, "err", err)
		} else {
			log.Info("Attempting to clone repository", "repoDir", repoDir, "repoURL", method.repoURL)
			err = cloneRepository(ctx, repoDir, method.repoURL, method.auth)
			if err == nil {
				log.Info("Successfully cloned repository", "repoDir", repoDir, "repoURL", method.repoURL)
				return nil
			}
			log.Warn("Failed to clone repository", "repoDir", repoDir, "repoURL", method.repoURL, "err", err)
		}
	}
	return fmt.Errorf("failed to clone/update repository %s, or no appropriate authentication method for repository", repoID)
}

func updateRepository(ctx context.Context, repoDir string, auth transport.AuthMethod) error {
	repo, err := git.PlainOpen(repoDir)
	if err != nil {
		return fmt.Errorf("unable to open Git repository %s: %v", repoDir, err)
	}
	wt, err := repo.Worktree()
	if err != nil {
		return fmt.Errorf("unable to open worktree for Git repository %s: %v", repoDir, err)
	}
	err = wt.PullContext(ctx, &git.PullOptions{
		Auth:       auth,
		RemoteName: "origin",
		Progress:   os.Stdout,
	})
	if err != nil && err != git.NoErrAlreadyUpToDate {
		return fmt.Errorf("failed to pull remote changes for %s: %v", repoDir, err)
	}
	return nil
}

func cloneRepository(ctx context.Context, repoDir, repoURL string, auth transport.AuthMethod) error {
	if err := os.MkdirAll(repoDir, 0o755); err != nil {
		return fmt.Errorf("failed to create repository directory %s: %v", repoDir, err)
	}
	_, err := git.PlainCloneContext(ctx, repoDir, false, &git.CloneOptions{
		Auth:       auth,
		URL:        repoURL,
		Progress:   os.Stdout,
		RemoteName: "origin",
	})
	if err != nil {
		return fmt.Errorf("failed to clone repository %s into %s: %v", repoURL, repoDir, err)
	}
	return nil
}
