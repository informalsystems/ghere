package ghere

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
)

// LocalCollection captures information about, and facilitates access to, local
// copies of GitHub repositories.
type LocalCollection struct {
	// Repositories is a list of specific repositories to fetch locally.
	Repositories []*LocalRepository `json:"repositories"`

	configFile string `json:"-"`
	rootPath   string `json:"-"`
}

func LoadOrCreateLocalCollection(configFile string) (*LocalCollection, error) {
	coll := &LocalCollection{}
	if err := readJSONFileOrEmpty(configFile, coll); err != nil {
		return nil, err
	}
	coll.configFile = configFile
	coll.rootPath = filepath.Dir(configFile)
	if err := coll.Save(); err != nil {
		return nil, err
	}
	return coll, nil
}

func (c *LocalCollection) Save() error {
	if err := writeJSONFile(c.configFile, c, true); err != nil {
		return fmt.Errorf("failed to write collection file: %v", err)
	}
	return nil
}

func (c *LocalCollection) NewFromPath(path string) (*LocalRepository, error) {
	parts := strings.Split(strings.TrimSpace(path), "/")
	if len(parts) == 0 || len(parts) > 2 {
		return nil, fmt.Errorf("invalid GitHub repository path: %s", parts)
	}
	for _, part := range parts {
		for _, r := range part {
			switch {
			case r == ' ' || r == '-' || r == '_' || r == '.':
			case r >= '0' && r <= '9':
			case r >= 'A' && r <= 'Z':
			case r >= 'a' && r <= 'z':
			default:
				return nil, fmt.Errorf("invalid character in path %s: %c", path, r)
			}
		}
	}
	for _, repo := range c.Repositories {
		if repo.Owner == parts[0] && repo.Name == parts[1] {
			return nil, &ErrRepositoryAlreadyExists{Owner: repo.Owner, Name: repo.Name}
		}
	}
	repo := &LocalRepository{
		Owner: parts[0],
		Name:  parts[1],
	}
	c.Repositories = append(c.Repositories, repo)
	return repo, nil
}

func (c *LocalCollection) Fetch(ctx context.Context, cfg *FetchConfig, log Logger) error {
	fetchers := []fetcher{}
	for _, repo := range c.Repositories {
		fetchers = append(fetchers, newRepoFetcher(c.rootPath, repo.Owner, repo.Name))
	}
	return fetchRecursively(ctx, cfg, fetchers, log)
}

type LocalRepository struct {
	Owner string `json:"owner"`
	Name  string `json:"name"`
}
