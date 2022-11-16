package ghere

import (
	"fmt"
	"path/filepath"
)

func repoPath(rootPath, owner, name string) string {
	return filepath.Join(rootPath, owner, name)
}

func repoDetailPath(rootPath, owner, name string) string {
	return filepath.Join(repoPath(rootPath, owner, name), DETAIL_FILENAME)
}

func repoPullRequestsPath(rootPath, owner, name string) string {
	return filepath.Join(repoPath(rootPath, owner, name), "pull-requests")
}

func repoCodePath(rootPath, owner, name string) string {
	return filepath.Join(repoPath(rootPath, owner, name), "code")
}

func pullRequestPath(rootPath, owner, name string, prNum int) string {
	return filepath.Join(repoPullRequestsPath(rootPath, owner, name), fmt.Sprintf("%.6d", prNum))
}

func pullRequestDetailPath(rootPath, owner, name string, prNum int) string {
	return filepath.Join(pullRequestPath(rootPath, owner, name, prNum), DETAIL_FILENAME)
}

// Path for all reviews for a specific pull request.
func pullRequestReviewsPath(rootPath, owner, name string, prNum int) string {
	return filepath.Join(pullRequestPath(rootPath, owner, name, prNum), "reviews")
}

func pullRequestCommentsPath(rootPath, owner, name string, prNum int) string {
	return filepath.Join(pullRequestPath(rootPath, owner, name, prNum), "comments")
}

// Path for a single review for a specific pull request.
func pullRequestReviewPath(rootPath, owner, name string, prNum int, reviewID int64) string {
	return filepath.Join(pullRequestPath(rootPath, owner, name, prNum), fmt.Sprintf("%d", reviewID))
}

func pullRequestReviewDetailPath(rootPath, owner, name string, prNum int, reviewID int64) string {
	return filepath.Join(pullRequestReviewPath(rootPath, owner, name, prNum, reviewID), DETAIL_FILENAME)
}

func pullRequestCommentPath(rootPath, owner, name string, prNum int, commentID int64) string {
	return filepath.Join(pullRequestCommentsPath(rootPath, owner, name, prNum), fmt.Sprintf("%d", commentID))
}

func reviewCommentPath(rootPath, owner, name string, prNum int, reviewID, commentID int64) string {
	return filepath.Join(pullRequestReviewPath(rootPath, owner, name, prNum, reviewID), fmt.Sprintf("%d.json", commentID))
}
