package ghere

import (
	"context"
	"fmt"
	"time"

	"github.com/google/go-github/v48/github"
)

func retryWithTimeout(ctx context.Context, retries int, retryTimeout time.Duration, log Logger, fn func(cx context.Context) (*github.Response, error)) (*github.Response, error) {
	for attempt := 0; attempt < retries; attempt++ {
		attemptCtx, cancelAttempt := context.WithCancel(ctx)
		resChan := make(chan *github.Response)
		errChan := make(chan error)
		go func() {
			defer cancelAttempt()
			res, err := fn(attemptCtx)
			if err != nil {
				errChan <- err
				return
			}
			resChan <- res
		}()
		select {
		case res := <-resChan:
			return res, nil
		case err := <-errChan:
			return nil, err
		case <-time.After(retryTimeout):
			log.Warn("Timed out while attempting GitHub request; retrying", "timeout", retryTimeout.String(), "attempt", attempt+1, "retries", retries)
			cancelAttempt()
		}
	}
	return nil, fmt.Errorf("failed to execute GitHub request %d times", retries)
}

type GitHubResponder func(ctx context.Context) (*github.Response, error)

func rateLimited(ctx context.Context, retries int, retryTimeout time.Duration, log Logger, ghr GitHubResponder) error {
	var res *github.Response
	var err error

	for {
		res, err = retryWithTimeout(ctx, retries, retryTimeout, log, ghr)
		if err == nil {
			break
		}
		if res.Rate.Remaining > 0 {
			return err
		}
		log.Warn("GitHub rate limit hit, waiting until reset time", "limit", res.Rate.Limit, "reset", res.Rate.Reset.Local().String())
		time.Sleep(time.Until(res.Rate.Reset.Time) + time.Second)
	}
	log.Debug("Rate limiting", "limit", res.Rate.Limit, "remaining", res.Rate.Remaining)
	return nil
}

// GitHubPaginatedResponder takes a page number and returns a GitHub response
// (from which we determine whether we are being rate-limited), a flag
// indicating whether we are done fetching pages, and optionally an error.
type GitHubPaginatedResponder func(ctx context.Context, page int) (*github.Response, bool, error)

func rateLimitedPaginated(ctx context.Context, startPage, retries int, retryTimeout time.Duration, log Logger, ghr GitHubPaginatedResponder) error {
	done := false
	for page := startPage; !done; page++ {
		err := rateLimited(ctx, retries, retryTimeout, log, func(cx context.Context) (res *github.Response, err error) {
			res, done, err = ghr(cx, page)
			return
		})
		if err != nil {
			return err
		}
	}
	return nil
}
