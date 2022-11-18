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
		log.Info("Fetching pull request comments", "repo", cf.repo.String(), "pr", pr.GetNumber())
		err := rateLimitedPaginated(
			ctx,
			1,
			cfg.RequestRetries,
			cfg.RequestTimeout,
			log,
			func(cx context.Context, pg int) (res *github.Response, done bool, err error) {
				var comments []*github.PullRequestComment
				comments, res, err = cfg.Client.PullRequests.ListComments(
					cx,
					cf.repo.GetOwner(),
					cf.repo.GetName(),
					pr.GetNumber(),
					&github.PullRequestListCommentsOptions{
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
				for _, ghComment := range comments {
					comment := &PullRequestComment{
						Comment: ghComment,
					}
					if err = comment.Save(cf.rootPath, cf.repo, pr.GetNumber(), cfg.PrettyJSON); err != nil {
						return
					}
				}
				done = len(comments) < DEFAULT_PER_PAGE
				return
			})
		if err != nil {
			return nil, fmt.Errorf("failed to fetch pull request %d comments for repo %s: %v", pr.GetNumber(), cf.repo, err)
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
		log.Info("Fetching pull request review comments", "repo", cf.repo.String(), "pr", review.PullRequestNumber, "reviewID", review.Review.GetID())
		// There shouldn't really be that many review comments on each review,
		// so fetching them from page 1 should be relatively optimal.
		err := rateLimitedPaginated(
			ctx,
			1,
			cfg.RequestRetries,
			cfg.RequestTimeout,
			log,
			func(cx context.Context, pg int) (res *github.Response, done bool, err error) {
				var comments []*github.PullRequestComment
				comments, res, err = cfg.Client.PullRequests.ListReviewComments(
					cx,
					cf.repo.GetOwner(),
					cf.repo.GetName(),
					review.PullRequestNumber,
					review.Review.GetID(),
					&github.ListOptions{
						Page:    pg,
						PerPage: DEFAULT_PER_PAGE,
					},
				)
				if err != nil {
					return
				}
				for _, ghComment := range comments {
					comment := &PullRequestComment{
						Comment: ghComment,
					}
					if err = comment.SaveForReview(cf.rootPath, cf.repo, review.PullRequestNumber, review.Review.GetID(), cfg.PrettyJSON); err != nil {
						return
					}
				}
				done = len(comments) < DEFAULT_PER_PAGE
				return
			})
		if err != nil {
			return nil, err
		}

		review.LastCommentsFetch = time.Now()
		if err := review.Save(cf.rootPath, cf.repo, cfg.PrettyJSON); err != nil {
			return nil, err
		}
	}
	return nil, nil
}
