package ghere

import (
	"github.com/google/go-github/v48/github"
)

type IssueComment struct {
	Comment *github.IssueComment `json:"comment"`
}
