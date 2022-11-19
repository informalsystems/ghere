package ghere

import (
	"context"
	"fmt"
	"time"

	"github.com/google/go-github/v48/github"
)

type Label struct {
	Label *github.Label
}

func LoadLabel(rootPath string, repo *Repository, labelID int64, mustExist bool) (*Label, error) {
	var err error
	label := &Label{}
	path := repoLabelPath(rootPath, repo.GetOwner(), repo.GetName(), labelID)
	if mustExist {
		err = readJSONFile(path, label)
	} else {
		err = readJSONFileOrEmpty(path, label)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to read repository label file: %v", err)
	}
	return label, nil
}

func (l *Label) Save(rootPath string, repo *Repository, prettyJSON bool) error {
	path := repoLabelPath(rootPath, repo.GetOwner(), repo.GetName(), l.Label.GetID())
	if err := writeJSONFile(path, l, prettyJSON); err != nil {
		return fmt.Errorf("failed to write repository label file: %v", err)
	}
	return nil
}

type labelsFetcher struct {
	rootPath string
	repo     *Repository
}

var _ fetcher = (*labelsFetcher)(nil)

func newLabelsFetcher(rootPath string, repo *Repository) *labelsFetcher {
	return &labelsFetcher{
		rootPath: rootPath,
		repo:     repo,
	}
}

func (f *labelsFetcher) fetch(ctx context.Context, cfg *FetchConfig, log Logger) ([]fetcher, error) {
	var labels []*github.Label
	var err error
	done := false
	for page := 1; !done; page++ {
		labels, done, err = cfg.Client.ListRepositoryLabels(
			ctx,
			f.repo.GetOwner(),
			f.repo.GetName(),
			page,
		)
		if err != nil {
			return nil, err
		}
		for _, ghLabel := range labels {
			label := &Label{
				Label: ghLabel,
			}
			if err := label.Save(f.rootPath, f.repo, cfg.PrettyJSON); err != nil {
				return nil, err
			}
		}
	}
	f.repo.LastLabelsFetch = time.Now()
	if err := f.repo.Save(f.rootPath, cfg.PrettyJSON); err != nil {
		return nil, err
	}
	return nil, nil
}
