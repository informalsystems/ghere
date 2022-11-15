package ghere

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/transport/ssh"
)

func cloneOrUpdateRepository(ctx context.Context, repoDir, repoURL, privKeyFile, privKeyFilePassword string) error {
	exists, err := dirExists(filepath.Join(repoDir, ".git"))
	if err != nil {
		return err
	}
	if exists {
		return updateRepository(ctx, repoDir, privKeyFile, privKeyFilePassword)
	}
	return cloneRepository(ctx, repoDir, repoURL, privKeyFile, privKeyFilePassword)
}

func updateRepository(ctx context.Context, repoDir, privKeyFile, privKeyFilePassword string) error {
	pubKeys, err := makeGitAuth(privKeyFile, privKeyFilePassword)
	if err != nil {
		return err
	}
	repo, err := git.PlainOpen(repoDir)
	if err != nil {
		return fmt.Errorf("unable to open Git repository %s: %v", repoDir, err)
	}
	wt, err := repo.Worktree()
	if err != nil {
		return fmt.Errorf("unable to open worktree for Git repository %s: %v", repoDir, err)
	}
	err = wt.PullContext(ctx, &git.PullOptions{
		Auth:       pubKeys,
		RemoteName: "origin",
		Progress:   os.Stdout,
	})
	if err != nil && err != git.NoErrAlreadyUpToDate {
		return fmt.Errorf("failed to pull remote changes for %s: %v", repoDir, err)
	}
	return nil
}

func cloneRepository(ctx context.Context, repoDir, repoURL, privKeyFile, privKeyFilePassword string) error {
	if err := os.MkdirAll(repoDir, 0o755); err != nil {
		return fmt.Errorf("failed to create repository directory %s: %v", repoDir, err)
	}
	pubKeys, err := makeGitAuth(privKeyFile, privKeyFilePassword)
	if err != nil {
		return err
	}
	_, err = git.PlainCloneContext(ctx, repoDir, false, &git.CloneOptions{
		Auth:       pubKeys,
		URL:        repoURL,
		Progress:   os.Stdout,
		RemoteName: "origin",
	})
	if err != nil {
		return fmt.Errorf("failed to clone repository %s into %s: %v", repoURL, repoDir, err)
	}
	return nil
}

func makeGitAuth(privKeyFile, privKeyFilePassword string) (*ssh.PublicKeys, error) {
	exists, err := fileExists(privKeyFile)
	if err != nil {
		return nil, fmt.Errorf("unable to access private key file %s: %v", privKeyFile, err)
	}
	if !exists {
		return nil, fmt.Errorf("private key file %s does not exist", privKeyFile)
	}
	pubKeys, err := ssh.NewPublicKeysFromFile("git", privKeyFile, privKeyFilePassword)
	if err != nil {
		return nil, fmt.Errorf("unable to generate public keys from %s: %v", privKeyFile, err)
	}
	return pubKeys, nil
}
