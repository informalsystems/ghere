package ghere

import (
	"context"
	"fmt"
	"time"

	"github.com/google/go-github/v48/github"
)

type IssueComment struct {
	Comment *github.IssueComment `json:"comment"`
}

type issueCommentsFetcher struct {
	rootPath string
	repo     *Repository
	issues   []*Issue
}

var _ fetcher = (*issueCommentsFetcher)(nil)

func newIssueCommentsFetcher(rootPath string, repo *Repository, issues []*Issue) *issueCommentsFetcher {
	return &issueCommentsFetcher{
		rootPath: rootPath,
		repo:     repo,
		issues:   issues,
	}
}

func (f *issueCommentsFetcher) fetch(ctx context.Context, cfg *FetchConfig, log Logger) ([]fetcher, error) {
	for _, issue := range f.issues {
		log.Info("Fetching comments for issue", "repo", f.repo.String(), "issue", issue.GetNumber())
		err := rateLimitedPaginated(
			ctx,
			1,
			cfg.RequestRetries,
			cfg.RequestTimeout,
			log,
			func(cx context.Context, pg int) (res *github.Response, done bool, err error) {
				var comments []*github.IssueComment
				sortParam := "created"
				dirParam := "asc"
				comments, res, err = cfg.Client.Issues.ListComments(
					cx,
					f.repo.GetOwner(),
					f.repo.GetName(),
					issue.GetNumber(),
					&github.IssueListCommentsOptions{
						Sort:      &sortParam,
						Direction: &dirParam,
						ListOptions: github.ListOptions{
							Page:    pg,
							PerPage: DEFAULT_PER_PAGE,
						},
					},
				)
				if err != nil {
					return
				}
				for _, ghComment := range comments {
					comment := &IssueComment{
						Comment: ghComment,
					}
					commentPath := issueCommentPath(
						f.rootPath,
						f.repo.GetOwner(),
						f.repo.GetName(),
						issue.GetNumber(),
						ghComment.GetID(),
					)
					if err = writeJSONFile(commentPath, comment, cfg.PrettyJSON); err != nil {
						err = fmt.Errorf("failed to write issue %d for repo %s, comment %s: %v", issue.GetNumber(), f.repo, commentPath, err)
						return
					}
				}
				done = len(comments) < DEFAULT_PER_PAGE
				return
			})
		if err != nil {
			return nil, err
		}

		issue.LastCommentsFetch = time.Now()
		idf := issueDetailPath(f.rootPath, f.repo.GetOwner(), f.repo.GetName(), issue.GetNumber())
		if err := writeJSONFile(idf, issue, cfg.PrettyJSON); err != nil {
			return nil, fmt.Errorf("failed to update last issue comments fetch time for issue %d in repo %s in %s: %v", issue.GetNumber(), f.repo, idf, err)
		}
	}
	return nil, nil
}
