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

## Git Workflow

Don't push to main, use feature branches instead.
