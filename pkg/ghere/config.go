package ghere

import (
	"time"

	"github.com/google/go-github/v48/github"
)

const (
	DEFAULT_PER_PAGE      int           = 100
	CONFIG_FILE_NAME      string        = "ghere.json"
	DETAIL_FILENAME       string        = "detail.json"
	FETCH_STATUS_FILENAME string        = "fetch-status.json"
	FULL_UPDATE_INTERVAL  time.Duration = 24 * time.Hour
)

// FetchConfig provides our configuration for all fetch operations.
type FetchConfig struct {
	Client                 *github.Client
	SSHPrivKeyFile         string
	SSHPrivKeyFilePassword string
	Concurrency            int
	RequestTimeout         time.Duration
	GitTimeout             time.Duration
	PrettyJSON             bool
}
