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
				labelFile := repoLabelPath(f.rootPath, f.repo.GetOwner(), f.repo.GetName(), ghLabel.GetID())
				if err = writeJSONFile(labelFile, label, cfg.PrettyJSON); err != nil {
					err = fmt.Errorf("failed to write label file for repository %s: %v", labelFile, err)
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
	rdf := repoDetailPath(f.rootPath, f.repo.GetOwner(), f.repo.GetName())
	if err := writeJSONFile(rdf, f.repo, cfg.PrettyJSON); err != nil {
		return nil, fmt.Errorf("failed to update last labels fetch time in repository details file %s: %v", rdf, err)
	}
	return nil, nil
}
