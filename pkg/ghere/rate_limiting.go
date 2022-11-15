package ghere

import (
	"time"

	"github.com/google/go-github/v48/github"
)

type GitHubResponder func() (*github.Response, error)

func rateLimited(log Logger, ghr GitHubResponder) error {
	var res *github.Response
	var err error

	for {
		res, err = ghr()
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
type GitHubPaginatedResponder func(page int) (*github.Response, bool, error)

func rateLimitedPaginated(startPage int, log Logger, ghr GitHubPaginatedResponder) error {
	done := false
	for page := startPage; !done; page++ {
		err := rateLimited(log, func() (res *github.Response, err error) {
			res, done, err = ghr(page)
			return
		})
		if err != nil {
			return err
		}
	}
	return nil
}
