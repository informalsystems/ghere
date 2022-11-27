# Changelog

## v0.2.0

*Nov 27, 2022*

- Change default `add` behavior to notify if a repository already exists instead
  of exiting with an error. This can now be turned into an error by supplying
  the `--fail-on-exists` flag when calling `add`.
- Change default `fetch` behavior to continue fetching other repositories, in
  spite of failures. This can be overridden by supplying the `--fail-fast` flag
  to the `fetch` command.

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
