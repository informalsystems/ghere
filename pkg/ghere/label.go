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
	log.Info("Fetching labels for repository", "repo", f.repo.String())
	err := rateLimitedPaginated(
		ctx,
		1,
		cfg.RequestRetries,
		cfg.RequestTimeout,
		log,
		func(cx context.Context, pg int) (res *github.Response, done bool, err error) {
			var labels []*github.Label
			labels, res, err = cfg.Client.Issues.ListLabels(
				cx,
				f.repo.GetOwner(),
				f.repo.GetName(),
				&github.ListOptions{
					Page:    pg,
					PerPage: DEFAULT_PER_PAGE,
				},
			)
			if err != nil {
				err = fmt.Errorf("failed to fetch labels for repository %s: %v", f.repo, err)
				return
			}
			for _, ghLabel := range labels {
				label := &Label{
					Label: ghLabel,
				}
				if err = label.Save(f.rootPath, f.repo, cfg.PrettyJSON); err != nil {
					return
				}
			}
			done = len(labels) < DEFAULT_PER_PAGE
			return
		})
	if err != nil {
		return nil, err
	}
	f.repo.LastLabelsFetch = time.Now()
	if err := f.repo.Save(f.rootPath, cfg.PrettyJSON); err != nil {
		return nil, err
	}
	return nil, nil
}
