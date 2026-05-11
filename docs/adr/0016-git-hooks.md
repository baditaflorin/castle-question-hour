# 0016 — Local git hooks (no GitHub Actions)

- Status: accepted
- Date: 2026-05-11

## Decision

Plain shell hooks under `.githooks/`, wired via `core.hooksPath` (`make install-hooks`). No
external tooling required for contributors — just the existing toolchain plus `gitleaks`.

| Hook         | Checks                                                                            |
|--------------|-----------------------------------------------------------------------------------|
| `pre-commit` | `gofmt`, `go vet`, `golangci-lint`, frontend `eslint`/`prettier`/`tsc`, `gitleaks` |
| `commit-msg` | Conventional Commits validator                                                     |
| `pre-push`   | `make test`, `make build` (verifies `docs/index.html` exists), `make smoke` (skip with `SKIP_SMOKE=1`) |
| `post-merge` | Regenerate OpenAPI TS client                                                       |
| `post-checkout` | Same as post-merge                                                              |

All hooks are idempotent and re-runnable manually: `make hooks-pre-commit`, etc.

## Why not lefthook

Lefthook is fine but adds a binary install step. Plain hooks are zero-dep and easier to audit.

## Why not GitHub Actions

§17 forbids it. The constraint is satisfied: every check runs on the contributor's machine.
