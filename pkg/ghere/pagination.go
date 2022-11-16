package ghere

import (
	"fmt"
	"path/filepath"
	"sort"
)

func paginatedItemsStartPage(pattern string, isOutdated func(string) (bool, error)) (int, error) {
	filenames, err := filepath.Glob(pattern)
	if err != nil {
		return 0, fmt.Errorf("failed to look up local items %s: %v", pattern, err)
	}
	sort.Strings(filenames)
	startPage := (len(filenames) / DEFAULT_PER_PAGE) + 1
	for _, fn := range filenames {
		outdated, err := isOutdated(fn)
		if err != nil {
			return 0, fmt.Errorf("failed to establish whether file %s is outdated: %v", fn, err)
		}
		if outdated {
			startPage = 1
			break
		}
	}
	return startPage, nil
}
