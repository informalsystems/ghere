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

func LoadIssue(rootPath string, repo *Repository, issueNum int, mustExist bool) (*Issue, error) {
	path := issueDetailPath(rootPath, repo.GetOwner(), repo.GetName(), issueNum)
	return LoadIssueDirect(path, mustExist)
}

func LoadIssueDirect(path string, mustExist bool) (*Issue, error) {
	var err error
	issue := &Issue{}
	if mustExist {
		err = readJSONFile(path, issue)
	} else {
		err = readJSONFileOrEmpty(path, issue)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to read issue detail file: %v", err)
	}
	return issue, nil
}

func (i *Issue) Save(rootPath string, repo *Repository, prettyJSON bool) error {
	path := issueDetailPath(rootPath, repo.GetOwner(), repo.GetName(), i.GetNumber())
	if err := writeJSONFile(path, i, prettyJSON); err != nil {
		return fmt.Errorf("failed to write issue detail file: %v", err)
	}
	return nil
}

// GetNumber is a shortcut for accessing the inner `Issue.GetNumber()` method.
func (i *Issue) GetNumber() int {
	return i.Issue.GetNumber()
}

func (i *Issue) MustUpdateComments() bool {
	return !i.Issue.IsPullRequest() && i.Issue.GetUpdatedAt().After(i.LastCommentsFetch)
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
		issue, err := LoadIssueDirect(fn, true)
		if err != nil {
			return false, err
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
				var issue *Issue
				issue, err = LoadIssue(f.rootPath, f.repo, ghIssue.GetNumber(), false)
				if err != nil {
					return
				}
				issue.Issue = ghIssue
				issue.LastDetailFetch = time.Now()
				if err = issue.Save(f.rootPath, f.repo, cfg.PrettyJSON); err != nil {
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

	f.repo.LastIssuesFetch = time.Now()
	if err := f.repo.Save(f.rootPath, cfg.PrettyJSON); err != nil {
		return nil, err
	}

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
		issue, err := LoadIssueDirect(fn, true)
		if err != nil {
			return nil, err
		}
		if issue.MustUpdateComments() {
			fetchComments = append(fetchComments, issue)
		}
	}

	fetchers := []fetcher{}
	if len(fetchComments) > 0 {
		fetchers = append(fetchers, newIssueCommentsFetcher(f.rootPath, f.repo, fetchComments))
	}

	return fetchers, nil
}
