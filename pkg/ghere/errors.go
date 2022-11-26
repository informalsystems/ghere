package ghere

import "fmt"

// ErrRepositoryAlreadyExists is returned from a call that attempts to create a
// repository, but that repository already exists.
type ErrRepositoryAlreadyExists struct {
	Owner string
	Name  string
}

var _ error = (*ErrRepositoryAlreadyExists)(nil)

func (e *ErrRepositoryAlreadyExists) Error() string {
	return fmt.Sprintf("repository already exists: %s/%s", e.Owner, e.Name)
}
