package ghere

import "context"

type fetcher interface {
	fetch(ctx context.Context, cfg *FetchConfig, log Logger) ([]fetcher, error)
}

func fetchRecursively(ctx context.Context, cfg *FetchConfig, fetchers []fetcher, log Logger) error {
	for _, fetcher := range fetchers {
		subFetchers, err := fetcher.fetch(ctx, cfg, log)
		if err != nil {
			return err
		}
		if len(subFetchers) > 0 {
			if err := fetchRecursively(ctx, cfg, subFetchers, log); err != nil {
				return err
			}
		}
	}
	return nil
}
