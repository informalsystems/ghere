# Changelog

## v0.1.1

*Nov 26, 2022*

- Add a flag `--warn-on-exists` to the `add` command to prevent the tool from
  erroring when a repository already exists. This is useful when used in
  conjunction with tools like Ansible in idempotently creating backup
  configurations.

## v0.1.0

*Nov 21, 2022*

ghere's first release! The tool is basically functional, allowing for fetching
of repositories and their associated code and metadata, but is not yet
well-tested. Consider it alpha quality at this point.
