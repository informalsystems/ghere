package ghere

import "github.com/google/go-github/v48/github"

type Issue struct {
	Issue *github.Issue `json:"issue"`
}
