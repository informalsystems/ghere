package ghere_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/google/go-github/v48/github"
	"github.com/informalsystems/ghere/pkg/ghere"
	"github.com/stretchr/testify/assert"
)

func TestCollectionFetching(t *testing.T) {
	log := ghere.NewNoopLogger()
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, ghere.CONFIG_FILE_NAME)
	coll, err := ghere.LoadOrCreateLocalCollection(configFile)
	assert.NoError(t, err)

	owner := "org"
	name := "repo"
	coll.Repositories = append(coll.Repositories, &ghere.LocalRepository{
		Owner: owner,
		Name:  name,
	})
	err = coll.Save()
	assert.NoError(t, err)
	assert.FileExists(t, configFile)

	repoID := owner + "/" + name
	mockClient := &MockGitHubClient{
		Repositories: map[string]*github.Repository{
			repoID: {
				Owner: &github.User{
					Login: &owner,
				},
				Name: &name,
			},
		},
	}

	cfg := &ghere.FetchConfig{
		Client:             mockClient,
		CredentialProvider: &MockGitHubCredentialProvider{},
		RepoUpdater:        &MockGitHubRepositoryUpdater{},
		PrettyJSON:         true,
	}

	err = coll.Fetch(context.Background(), cfg, log)
	assert.NoError(t, err)

	localRepoDir := filepath.Join(tmpDir, owner, name)
	localDetailsFile := filepath.Join(localRepoDir, ghere.DETAIL_FILENAME)
	assert.DirExists(t, localRepoDir)
	assert.FileExists(t, localDetailsFile)

	repo := &ghere.Repository{}
	err = ghere.ReadJSONFile(localDetailsFile, repo)
	assert.NoError(t, err)
	assert.Equal(t, mockClient.Repositories[repoID], repo.Repository)
}
