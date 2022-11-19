package ghere

import (
	"time"
)

const (
	DEFAULT_PER_PAGE int    = 100
	CONFIG_FILE_NAME string = "ghere.json"
	DETAIL_FILENAME  string = "detail.json"
)

// FetchConfig provides our configuration for all fetch operations.
type FetchConfig struct {
	Client                 GitHubClient
	SSHPrivKeyFile         string
	SSHPrivKeyFilePassword string
	GitTimeout             time.Duration
	PrettyJSON             bool
}
