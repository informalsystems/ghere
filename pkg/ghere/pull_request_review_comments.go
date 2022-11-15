package ghere

import "github.com/google/go-github/v48/github"

type PullRequestReviewComment struct {
	Comment *github.PullRequestComment `json:"comment"`
}
