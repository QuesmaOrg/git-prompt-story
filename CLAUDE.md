# CLAUDE.md

## UI Library

This project uses [Bubble Tea](https://github.com/charmbracelet/bubbletea) for rich terminal UI.

## Testing

Run end-to-end tests with:

```bash
e2e/run-tests.sh
```

Note: Tests can be slow as they use Docker containers.

## After Making Changes

After making progress on the codebase, offer to reinstall the global `git-prompt-story`:

```bash
go install .
```

Note: Local builds will show version "dev". Release builds get version from git tags via GoReleaser.

## Previewing PR Summary Changes

When modifying PR summary output (`internal/ci/summary.go`), offer to preview changes locally before merging:

1. Rebuild: `go build .`
2. Run against the PR branch: `./git-prompt-story pr summary "origin/main..HEAD" --format=markdown | pbcopy`
3. Ask user paste as a GitHub PR comment to preview how the output will look in GitHub

## Git Workflow

Don't push to main, use feature branches instead.
