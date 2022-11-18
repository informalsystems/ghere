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

func LoadIssueComment(rootPath string, repo *Repository, issueNum int, commentID int64, mustExist bool) (*IssueComment, error) {
	path := issueCommentPath(rootPath, repo.GetOwner(), repo.GetName(), issueNum, commentID)
	return LoadIssueCommentDirect(path, mustExist)
}

func LoadIssueCommentDirect(path string, mustExist bool) (*IssueComment, error) {
	var err error
	comment := &IssueComment{}
	if mustExist {
		err = readJSONFile(path, comment)
	} else {
		err = readJSONFileOrEmpty(path, comment)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to read issue comment file: %v", err)
	}
	return comment, nil
}

func (c *IssueComment) Save(rootPath string, repo *Repository, issueNum int, prettyJSON bool) error {
	path := issueCommentPath(rootPath, repo.GetOwner(), repo.GetName(), issueNum, c.Comment.GetID())
	if err := writeJSONFile(path, c, prettyJSON); err != nil {
		return fmt.Errorf("failed to write issue comment file: %v", err)
	}
	return nil
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
					if err = comment.Save(f.rootPath, f.repo, issue.GetNumber(), cfg.PrettyJSON); err != nil {
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
		if err := issue.Save(f.rootPath, f.repo, cfg.PrettyJSON); err != nil {
			return nil, err
		}
	}
	return nil, nil
}
