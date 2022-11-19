package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/google/go-github/v48/github"
	"github.com/informalsystems/ghere/pkg/ghere"
	"github.com/spf13/cobra"
	"golang.org/x/oauth2"
)

type fetchCmd struct {
	*cobra.Command

	privKeyFile string
	reqRetries  uint
	reqTimeout  uint
	gitTimeout  uint
	pretty      bool
}

func newFetchCmd(root *rootCmd) *fetchCmd {
	cmd := &fetchCmd{}
	cmd.Command = &cobra.Command{
		Use:   "fetch",
		Short: "Fetch a local collection's repositories from GitHub",
		Example: `  # Set your GitHub personal access token to raise rate limits
  # See https://docs.github.com/en/rest/overview/resources-in-the-rest-api#rate-limiting
  # Note the space before the command - this prevents the shell from saving it
  # in your history
   export GITHUB_TOKEN="..."

  # Set your SSH key's password (if any)
   export SSH_PRIVKEY_PASSWORD="..."

  # Fetch all repositories
  ghere fetch`,
		RunE: func(c *cobra.Command, args []string) error {
			log := root.logger

			accessToken := os.Getenv("GITHUB_TOKEN")
			if len(accessToken) == 0 {
				log.Error("To fetch from GitHub, you must set the GITHUB_TOKEN environment variable")
				return errors.New("missing GITHUB_TOKEN environment variable")
			}

			sshPrivKeyPassword := os.Getenv("SSH_PRIVKEY_PASSWORD")

			log.Info("Loading local collection", "path", root.configFile)
			coll, err := ghere.LoadOrCreateLocalCollection(root.configFile)
			if err != nil {
				log.Error("Failed to load collection", "err", err)
				return err
			}
			ts := oauth2.StaticTokenSource(
				&oauth2.Token{
					AccessToken: accessToken,
				},
			)
			tc := oauth2.NewClient(c.Context(), ts)
			client := github.NewClient(tc)
			reqRetries := int(cmd.reqRetries)
			reqTimeout := time.Duration(cmd.reqTimeout) * time.Second
			cfg := &ghere.FetchConfig{
				Client:                 ghere.NewGitHubClient(client, reqRetries, reqTimeout, log),
				SSHPrivKeyFile:         cmd.privKeyFile,
				SSHPrivKeyFilePassword: sshPrivKeyPassword,
				GitTimeout:             time.Duration(cmd.gitTimeout) * time.Second,
				PrettyJSON:             cmd.pretty,
			}
			if err := coll.Fetch(c.Context(), cfg, log); err != nil {
				log.Error("Failed to sync from GitHub", "err", err)
				return err
			}
			return nil
		},
	}
	homeDir, err := os.UserHomeDir()
	if err != nil {
		panic(fmt.Sprintf("failed to obtain user's home directory: %v", err))
	}
	defaultPrivKeyPath := filepath.Join(homeDir, ".ssh", "id_rsa")
	cmd.Flags().StringVar(&cmd.privKeyFile, "priv-key", defaultPrivKeyPath, "path to the private key to use to clone Git repositories")
	cmd.Flags().UintVar(&cmd.reqRetries, "request-retries", 3, "how many times to retry requests to GitHub that timeout")
	cmd.Flags().UintVar(&cmd.reqTimeout, "request-timeout", 20, "timeout, in seconds, for each HTTP request")
	cmd.Flags().UintVar(&cmd.gitTimeout, "git-timeout", 120, "timeout, in seconds, for each Git repository clone/pull operation")
	cmd.Flags().BoolVar(&cmd.pretty, "pretty", false, "output pretty JSON instead of compact JSON")
	return cmd
}
