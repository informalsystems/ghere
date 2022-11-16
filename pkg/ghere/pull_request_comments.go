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
		err := rateLimitedPaginated(1, log, func(pg int) (res *github.Response, done bool, err error) {
			var comments []*github.PullRequestComment
			comments, res, err = cfg.Client.PullRequests.ListComments(
				ctx,
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
			for _, comment := range comments {
				c := &PullRequestComment{
					Comment: comment,
				}
				commentPath := pullRequestCommentPath(
					cf.rootPath,
					cf.repo.GetOwner(),
					cf.repo.GetName(),
					pr.GetNumber(),
					comment.GetID(),
				)
				if err = writeJSONFile(commentPath, c, cfg.PrettyJSON); err != nil {
					err = fmt.Errorf("failed to write pull request comment %s: %v", commentPath, err)
				}
			}
			done = len(comments) < DEFAULT_PER_PAGE
			return
		})
		if err != nil {
			return nil, fmt.Errorf("failed to fetch pull request %d comments for repo %s: %v", pr.GetNumber(), cf.repo, err)
		}

		pr.LastCommentsFetch = time.Now()
		prDetailFile := pullRequestDetailPath(
			cf.rootPath,
			cf.repo.GetOwner(),
			cf.repo.GetName(),
			pr.GetNumber(),
		)
		if err := writeJSONFile(prDetailFile, pr, cfg.PrettyJSON); err != nil {
			return nil, fmt.Errorf("failed to update last comments fetch time for pull request %d for repo %s: %v", pr.GetNumber(), cf.repo, err)
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
		// There shouldn't really be that many review comments on each review,
		// so fetching them from page 1 should be relatively optimal.
		err := rateLimitedPaginated(1, log, func(pg int) (res *github.Response, done bool, err error) {
			var comments []*github.PullRequestComment
			comments, res, err = cfg.Client.PullRequests.ListReviewComments(
				ctx,
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
			for _, comment := range comments {
				c := &PullRequestComment{
					Comment: comment,
				}
				commentPath := reviewCommentPath(
					cf.rootPath,
					cf.repo.GetOwner(),
					cf.repo.GetName(),
					review.PullRequestNumber,
					review.Review.GetID(),
					comment.GetID(),
				)
				if err = writeJSONFile(commentPath, c, cfg.PrettyJSON); err != nil {
					err = fmt.Errorf("failed to write review comment file %s: %v", commentPath, err)
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
		reviewDetailFile := pullRequestReviewDetailPath(
			cf.rootPath,
			cf.repo.GetOwner(),
			cf.repo.GetName(),
			review.PullRequestNumber,
			review.Review.GetID(),
		)
		if err := writeJSONFile(reviewDetailFile, review, cfg.PrettyJSON); err != nil {
			return nil, fmt.Errorf("failed to update last comments fetch time for review %s: %v", reviewDetailFile, err)
		}
	}
	return nil, nil
}
