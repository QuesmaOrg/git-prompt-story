# CLAUDE.md

## Testing

Run end-to-end tests with:

```bash
e2e/run-tests.sh
```

Note: Tests can be slow as they use Docker containers.

## After Making Changes

After making progress on the codebase, offer to reinstall the global `git-prompt-story`, bump version in `VERSION` file and do:

```bash
go install .
```
