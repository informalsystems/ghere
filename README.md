# ghere

GitHub repos, with issues, pull requests and comments, over _here_, on my local
machine. Pronounced "gear".

**NB: This is still a work-in-progress. Please expect substantial changes.**

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
# Initialize an empty local collection of repositories.
ghere init

# Add a local copy of https://github.com/org/repo. Does not sync anything at
# this point - just adds it to the local collection.
ghere add org/repo

# Fetch the code, metadata, plus all latest issues, pull requests and comments
# for all configured repositories. By default, this does not output pretty JSON.
ghere fetch

# Fetch the code, but prettifying the JSON before writing it to disk.
ghere fetch --pretty

# Increase output logging to debug level, and prettify the JSON output.
ghere fetch -v --pretty
```
