# Contributing

Thanks for poking at castle-question-hour.

## Ground rules

- **No GitHub Actions.** Every check runs locally via git hooks (see [.githooks/](.githooks)).
- **No secrets in git, ever.** `gitleaks` runs in `pre-commit`. The frontend never holds secrets.
- **Conventional Commits.** Enforced by `commit-msg` hook.
- **Small, focused commits.** Don't bundle unrelated changes.
- **ADR-first for non-trivial decisions.** Add a new file under `docs/adr/` *before* the code lands.

## Bootstrapping

```bash
make install-hooks
make dev
```

## Layout & style

- Go: idiomatic, `gofmt`/`goimports` clean, `golangci-lint` clean. Errors wrapped with `%w`. No globals.
- TS: strict mode, `eslint`/`prettier` clean, `tsc --noEmit` clean.
- One concern per file. No file >300 lines without a strong reason.

## Tests

- `make test` — unit, fast.
- `make test-integration` — integration, may spin compose.
- `make smoke` — end-to-end happy path.

PRs that break the pre-push hook will be sent back.

## Reporting bugs / requesting features

Open an issue with steps to reproduce or a concrete user story. Security issues — see
[SECURITY.md](SECURITY.md), not the issue tracker.
