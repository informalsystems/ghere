# ghere

GitHub repos, with issues, pull requests and comments, over _here_, on my local
machine. Pronounced "gear".

**NB: This is still a work-in-progress. Please expect substantial changes.**

## Usage

```bash
# Initialize an empty local collection of repositories.
ghere init

# Add a local copy of https://github.com/org/repo. Does not sync anything at
# this point - just adds it to the local collection.
ghere add org/repo

# Fetch the code, metadata, plus all latest issues, pull requests and comments
# for all configured repositories.
ghere fetch
```
