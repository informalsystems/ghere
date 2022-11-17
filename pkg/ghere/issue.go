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

// GetNumber is a shortcut for accessing the inner `Issue.GetNumber()` method.
func (i *Issue) GetNumber() int {
	return i.Issue.GetNumber()
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
		repoUpdatedRecently := f.repo.Repository.GetUpdatedAt().After(issue.LastDetailFetch)
		fetchedMoreThan24hAgo := time.Since(issue.LastDetailFetch) > 24*time.Hour
		return repoUpdatedRecently && fetchedMoreThan24hAgo, nil
	})
	if err != nil {
		return nil, err
	}
	err = rateLimitedPaginated(
		ctx,
		startPage,
		cfg.RequestRetries,
		cfg.RequestTimeout,
		log,
		func(cx context.Context, pg int) (res *github.Response, done bool, err error) {
			var issues []*github.Issue
			log.Info("Fetching page of issues", "repo", f.repo.String(), "page", pg)
			issues, res, err = cfg.Client.Issues.ListByRepo(
				cx,
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

	return f.makeCommentsFetcher(log)
}

func (f *issuesFetcher) makeCommentsFetcher(log Logger) ([]fetcher, error) {
	log.Info("Computing which issues' comments should be fetched", "repo", f.repo.String())
	fetchComments := []*Issue{}
	issuesPath := repoIssuesPath(f.rootPath, f.repo.GetOwner(), f.repo.GetName())
	pattern := filepath.Join(issuesPath, "*", DETAIL_FILENAME)
	issueDetailFiles, err := filepath.Glob(pattern)
	if err != nil {
		return nil, fmt.Errorf("failed to list issues' detail files from pattern %s: %v", pattern, err)
	}

	for _, fn := range issueDetailFiles {
		issue := &Issue{}
		if err := readJSONFile(fn, issue); err != nil {
			return nil, err
		}
		if !issue.Issue.IsPullRequest() && issue.Issue.GetUpdatedAt().After(issue.LastCommentsFetch) {
			fetchComments = append(fetchComments, issue)
		}
	}

	fetchers := []fetcher{}
	if len(fetchComments) > 0 {
		fetchers = append(fetchers, newIssueCommentsFetcher(f.rootPath, f.repo, fetchComments))
	}

	return fetchers, nil
}
