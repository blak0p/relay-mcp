# Contributing

Thanks for your interest in relay-mcp.

## How to contribute

1. Open an issue first to discuss the change you want to make.
2. Wait for the issue to get the `status:approved` label.
3. Fork the repo and create a branch from `main`.
4. Write tests for your change.
5. Run `go test ./...` and `go vet ./...` before committing.
6. Open a pull request referencing the issue (e.g. "Closes #123").
7. The PR will auto-assign `status:needs-review`. I'll review and apply `status:approved` when it's ready.

## Commit style

Conventional Commits, no scope parentheticals, no AI attribution:

```
feat: add run_command one-shot tool
fix: prevent race on session close
chore: bump creack/pty to v1.1.22
```

## Code conventions

- **Module layout**: namespace parent + sub-packages. Each package owns one responsibility.
- **Docs pair**: every package has both a `README.md` (GitHub) and a `doc.go` (`go doc`).
- **DRY pragmatic**: reuse when natural, don't force coupling between distinct concerns.

## Before opening a PR

- `go vet ./...` passes
- `go test -race -shuffle=on -count=1 ./...` passes
- PR body references the issue with `Closes #N`
- The referenced issue has `status:approved`
