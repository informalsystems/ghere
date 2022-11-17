package ghere

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	"github.com/google/go-github/v48/github"
)

type Issue struct {
	Issue *github.Issue `json:"issue"`

	LastDetailFetch   time.Time `json:"last_detail_fetch"`
	LastCommentsFetch time.Time `json:"last_comments_fetch"`
}

type issuesFetcher struct {
	rootPath string
	repo     *Repository
}

var _ fetcher = (*issuesFetcher)(nil)

func newIssuesFetcher(rootPath string, repo *Repository) *issuesFetcher {
	return &issuesFetcher{
		rootPath: rootPath,
		repo:     repo,
	}
}

func (f *issuesFetcher) fetch(ctx context.Context, cfg *FetchConfig, log Logger) ([]fetcher, error) {
	log.Info("Fetching issues for repository", "repo", f.repo.String())
	issuesPath := repoIssuesPath(f.rootPath, f.repo.GetOwner(), f.repo.GetName())
	pattern := filepath.Join(issuesPath, "*", DETAIL_FILENAME)
	startPage, err := paginatedItemsStartPage(pattern, func(fn string) (bool, error) {
		issue := &Issue{}
		if err := readJSONFile(fn, issue); err != nil {
			return false, fmt.Errorf("failed to read issue detail file %s: %v", fn, err)
		}
		return f.repo.Repository.GetUpdatedAt().After(issue.LastDetailFetch), nil
	})
	if err != nil {
		return nil, err
	}
	err = rateLimitedPaginated(startPage, log, func(pg int) (res *github.Response, done bool, err error) {
		var issues []*github.Issue
		log.Info("Fetching page of issues", "repo", f.repo.String(), "page", pg)
		issues, res, err = cfg.Client.Issues.ListByRepo(
			ctx,
			f.repo.GetOwner(),
			f.repo.GetName(),
			&github.IssueListByRepoOptions{
				State:     "all",
				Sort:      "created",
				Direction: "asc",
				ListOptions: github.ListOptions{
					Page:    pg,
					PerPage: DEFAULT_PER_PAGE,
				},
			},
		)
		if err != nil {
			return
		}
		for _, ghIssue := range issues {
			issue := &Issue{}
			idp := issueDetailPath(f.rootPath, f.repo.GetOwner(), f.repo.GetName(), ghIssue.GetNumber())
			if err = readJSONFileOrEmpty(idp, issue); err != nil {
				err = fmt.Errorf("failed to read issue detail file %s: %v", idp, err)
				return
			}
			issue.Issue = ghIssue
			issue.LastDetailFetch = time.Now()
			if err = writeJSONFile(idp, issue, cfg.PrettyJSON); err != nil {
				err = fmt.Errorf("failed to write issue detail file %s: %v", idp, err)
				return
			}
		}
		done = len(issues) < DEFAULT_PER_PAGE
		return
	})
	if err != nil {
		return nil, err
	}
	log.Info("Fetched all issues' details", "repo", f.repo.String())

	return nil, nil
}
