package ghere

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	"github.com/google/go-github/v48/github"
)

type PullRequest struct {
	PullRequest *github.PullRequest `json:"pull_request"`

	LastDetailFetch   time.Time `json:"last_detail_fetch"`
	LastReviewsFetch  time.Time `json:"last_reviews_fetch"`
	LastCommentsFetch time.Time `json:"last_comments_fetch"`
}

type pullRequestsFetcher struct {
	pullsPath string
	repo      *Repository
}

func newPullRequestsFetcher(pullsPath string, repo *Repository) *pullRequestsFetcher {
	return &pullRequestsFetcher{
		pullsPath: pullsPath,
		repo:      repo,
	}
}

func (pf *pullRequestsFetcher) fetch(ctx context.Context, cfg *FetchConfig, log Logger) ([]fetcher, error) {
	log.Info("Fetching pull requests for repository", "repo", pf.repo.String())
	pattern := filepath.Join(pf.pullsPath, "*", DETAIL_FILENAME)
	startPage, err := paginatedItemsStartPage(pattern, func(fn string) (bool, error) {
		pr := &PullRequest{}
		if err := readJSONFile(fn, pr); err != nil {
			return false, err
		}
		// If we haven't fetched this pull request's details in 24 hours,
		// consider it outdated.
		return time.Since(pr.LastDetailFetch) > FULL_UPDATE_INTERVAL, nil
	})
	if err != nil {
		return nil, err
	}
	// Pull request numbers whose reviews and comments must be fetched.
	fetchReviews := []int{}
	fetchComments := []int{}
	err = rateLimitedPaginated(startPage, log, func(pg int) (res *github.Response, done bool, err error) {
		var pulls []*github.PullRequest
		log.Info("Fetching page of pull requests", "page", pg)
		pulls, res, err = cfg.Client.PullRequests.List(ctx, pf.repo.GetOwner(), pf.repo.GetName(), &github.PullRequestListOptions{
			State:     "all",
			Sort:      "created",
			Direction: "asc",
			ListOptions: github.ListOptions{
				Page:    pg,
				PerPage: DEFAULT_PER_PAGE,
			},
		})
		if err != nil {
			return
		}
		for _, pull := range pulls {
			pr := &PullRequest{}
			prPath := filepath.Join(pf.pullsPath, fmt.Sprintf("%.6d", pull.GetNumber()), DETAIL_FILENAME)
			if err = readJSONFileOrEmpty(prPath, pr); err != nil {
				return
			}
			pr.PullRequest = pull
			pr.LastDetailFetch = time.Now()
			if err = writeJSONFile(prPath, pr, cfg.PrettyJSON); err != nil {
				return
			}
			if pull.GetUpdatedAt().After(pr.LastReviewsFetch) {
				fetchReviews = append(fetchReviews, pull.GetNumber())
			}
			if pull.GetUpdatedAt().After(pr.LastCommentsFetch) {
				fetchComments = append(fetchComments, pull.GetNumber())
			}
		}
		done = len(pulls) < DEFAULT_PER_PAGE
		return
	})
	if err != nil {
		return nil, err
	}
	log.Info("Fetched all pull requests' details", "repo", pf.repo.String())

	fetchers := []fetcher{}
	if len(fetchReviews) > 0 {
		fetchers = append(fetchers, newPullRequestReviewFetcher(
			pf.pullsPath,
			pf.repo,
			fetchReviews,
		))
	}
	if len(fetchComments) > 0 {
		fetchers = append(fetchers, newPullRequestCommentsFetcher(
			pf.pullsPath,
			pf.repo,
			fetchComments,
		))
	}

	return fetchers, nil
}
