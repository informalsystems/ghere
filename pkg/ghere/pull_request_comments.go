package ghere

import (
	"context"
	"fmt"
	"time"

	"github.com/google/go-github/v48/github"
)

type PullRequestComment struct {
	Comment *github.PullRequestComment `json:"comment"`
}

func LoadPullRequestComment(rootPath string, repo *Repository, prNum int, commentID int64, mustExist bool) (*PullRequestComment, error) {
	path := pullRequestCommentPath(
		rootPath,
		repo.GetOwner(),
		repo.GetName(),
		prNum,
		commentID,
	)
	return LoadPullRequestCommentDirect(path, mustExist)
}

func LoadPullRequestReviewComment(rootPath string, repo *Repository, prNum int, reviewID, commentID int64, mustExist bool) (*PullRequestComment, error) {
	path := reviewCommentPath(
		rootPath,
		repo.GetOwner(),
		repo.GetName(),
		prNum,
		reviewID,
		commentID,
	)
	return LoadPullRequestCommentDirect(path, mustExist)
}

func LoadPullRequestCommentDirect(path string, mustExist bool) (*PullRequestComment, error) {
	var err error
	pr := &PullRequestComment{}
	if mustExist {
		err = readJSONFile(path, pr)
	} else {
		err = readJSONFileOrEmpty(path, pr)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to read pull request comment file: %v", err)
	}
	return pr, nil
}

func (c *PullRequestComment) Save(rootPath string, repo *Repository, prNum int, prettyJSON bool) error {
	path := pullRequestCommentPath(
		rootPath,
		repo.GetOwner(),
		repo.GetName(),
		prNum,
		c.Comment.GetID(),
	)
	if err := writeJSONFile(path, c, prettyJSON); err != nil {
		return fmt.Errorf("failed to write pull request comment file: %v", err)
	}
	return nil
}

func (c *PullRequestComment) SaveForReview(rootPath string, repo *Repository, prNum int, reviewID int64, prettyJSON bool) error {
	path := reviewCommentPath(
		rootPath,
		repo.GetOwner(),
		repo.GetName(),
		prNum,
		reviewID,
		c.Comment.GetID(),
	)
	if err := writeJSONFile(path, c, prettyJSON); err != nil {
		return fmt.Errorf("failed to write pull request review comment: %v", err)
	}
	return nil
}

// Fetches all comments on pull requests - not just those associated with
// specific reviews.
type pullRequestCommentsFetcher struct {
	rootPath     string
	repo         *Repository
	pullRequests []*PullRequest
}

func newPullRequestCommentsFetcher(rootPath string, repo *Repository, pullRequests []*PullRequest) *pullRequestCommentsFetcher {
	return &pullRequestCommentsFetcher{
		rootPath:     rootPath,
		repo:         repo,
		pullRequests: pullRequests,
	}
}

func (cf *pullRequestCommentsFetcher) fetch(ctx context.Context, cfg *FetchConfig, log Logger) ([]fetcher, error) {
	for _, pr := range cf.pullRequests {
		var err error
		done := false
		for page := 1; !done; page++ {
			var comments []*github.PullRequestComment
			comments, done, err = cfg.Client.ListPullRequestComments(
				ctx,
				cf.repo.GetOwner(),
				cf.repo.GetName(),
				pr.GetNumber(),
				page,
			)
			if err != nil {
				return nil, err
			}
			for _, ghComment := range comments {
				comment := &PullRequestComment{
					Comment: ghComment,
				}
				if err := comment.Save(cf.rootPath, cf.repo, pr.GetNumber(), cfg.PrettyJSON); err != nil {
					return nil, err
				}
			}
		}
		pr.LastCommentsFetch = time.Now()
		if err := pr.Save(cf.rootPath, cf.repo, cfg.PrettyJSON); err != nil {
			return nil, err
		}
	}

	return nil, nil
}

// Fetches comments specific to pull request reviews.
type pullRequestReviewCommentsFetcher struct {
	rootPath string
	repo     *Repository
	reviews  []*PullRequestReview
}

func newPullRequestReviewCommentsFetcher(rootPath string, repo *Repository, reviews []*PullRequestReview) *pullRequestReviewCommentsFetcher {
	return &pullRequestReviewCommentsFetcher{
		rootPath: rootPath,
		repo:     repo,
		reviews:  reviews,
	}
}

func (cf *pullRequestReviewCommentsFetcher) fetch(ctx context.Context, cfg *FetchConfig, log Logger) ([]fetcher, error) {
	for _, review := range cf.reviews {
		var err error
		done := false
		for page := 1; !done; page++ {
			var comments []*github.PullRequestComment
			comments, done, err = cfg.Client.ListPullRequestReviewComments(
				ctx,
				cf.repo.GetOwner(),
				cf.repo.GetName(),
				review.PullRequestNumber,
				review.Review.GetID(),
				page,
			)
			if err != nil {
				return nil, err
			}
			for _, ghComment := range comments {
				comment := &PullRequestComment{
					Comment: ghComment,
				}
				if err := comment.SaveForReview(cf.rootPath, cf.repo, review.PullRequestNumber, review.Review.GetID(), cfg.PrettyJSON); err != nil {
					return nil, err
				}
			}
		}
		review.LastCommentsFetch = time.Now()
		if err := review.Save(cf.rootPath, cf.repo, cfg.PrettyJSON); err != nil {
			return nil, err
		}
	}
	return nil, nil
}
