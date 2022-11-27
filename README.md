# ghere

![GitHub release (latest SemVer)](https://img.shields.io/github/v/release/informalsystems/ghere)
[![Build](https://github.com/informalsystems/ghere/actions/workflows/build.yml/badge.svg)](https://github.com/informalsystems/ghere/actions/workflows/build.yml)
[![Linter](https://github.com/informalsystems/ghere/actions/workflows/lint.yml/badge.svg)](https://github.com/informalsystems/ghere/actions/workflows/lint.yml)
[![Test](https://github.com/informalsystems/ghere/actions/workflows/tests.yml/badge.svg)](https://github.com/informalsystems/ghere/actions/workflows/tests.yml)

GitHub repos, with issues, pull requests and comments, over _here_, on my local
machine. Pronounced "gear".

## Requirements

* Go 1.19+

## Installation

```bash
# First clone this repository. Then from the root of the repository:
make install
```

## Usage

See `ghere --help` for details.

```bash
# Set your GitHub personal access token to raise rate limits
# See https://docs.github.com/en/rest/overview/resources-in-the-rest-api#rate-limiting
# Note the space before the command - this prevents the shell from saving it
# in your history.
 export GITHUB_TOKEN="..."

# Set your SSH key's password (if any). This allows ghere to automatically pull
# Git repositories while authenticating you via SSH. Again, note the space
# before the command to prevent the shell from saving it to your history.
 export SSH_PRIVKEY_PASSWORD="..."

# Initialize an empty local collection of repositories.
ghere init

# Add a local copy of https://github.com/org/repo. Does not sync anything at
# this point - just adds it to the local collection. Idempotent.
ghere add org/repo

# The --fail-on-exists flag will cause ghere to exit with an error if the
# repository already exists in the collection, as opposed to its default
# behavior of notifying.
ghere add --fail-on-exists org/repo

# Fetch the code, metadata, plus all latest issues, pull requests and comments
# for all configured repositories. By default, this does not output pretty JSON.
ghere fetch

# Fetch the code, but prettifying the JSON before writing it to disk.
ghere fetch --pretty

# Increase output logging to debug level, and prettify the JSON output.
ghere fetch -v --pretty
```

## Features

- [ ] Fetch entire organizations
- [ ] Fetch projects
- [ ] Fetch teams
- [x] Fetch individual repositories (public and private, depending on personal
  access token privileges)
- [x] Fetch code (Git repository)
  - [x] Fetch code via SSH with SSH key support
  - [x] Fetch code via HTTPS
- [x] Fetch issues
  - [x] Fetch issue comments
- [x] Fetch pull requests
  - [x] Fetch pull request comments
  - [x] Fetch pull request reviews
    - [x] Fetch pull request review comments
- [ ] Fetch releases
- [x] Fetch repository labels
- [ ] Fetch milestones
- [ ] Fetch wikis
- [ ] Fetch gists
- [ ] Fetch related media (e.g. embedded images in issue/pull request
  descriptions and comments)
- [x] Handle GitHub rate limiting (when individual rate limits are hit, ghere
  automatically waits until the rate limit reset time to continue)
- [x] Handle request retries
- [x] Incremental update (tries to minimize the number of requests to the GitHub
  API)
