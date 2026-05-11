# 0010 — GitHub Pages publishing strategy

- Status: accepted
- Date: 2026-05-11

## Decision

- **Source**: `main` branch, **`/docs` folder**.
- **Build output dir**: `docs/`.
- **Base path**: `/castle-question-hour/` (set via Vite `base` from env `BASE_PATH`).
- **Public URL**: `https://baditaflorin.github.io/castle-question-hour/`.
- **SPA fallback**: copy `docs/index.html` → `docs/404.html` post-build so any client-routed URL
  returns the SPA shell.
- **Cache busting**: Vite emits hashed filenames (`assets/index-<hash>.js`). The unhashed
  `index.html` is the only re-fetched file.
- **Custom domain**: not configured in v1. If added, a `docs/CNAME` file holds the hostname and
  ADR 0010 is amended.
- The `.gitignore` keeps `docs/` *committed* but ignores the runtime data dump under `docs/data/`.

## Consequences

- Every release commit includes a fresh `docs/` rebuild. The `pre-push` hook ensures the build
  succeeds and `docs/index.html` exists before push.
- No `gh-pages` branch. No GitHub Actions. Pages is just "serve `/docs` from main".

## Operator runbook

```bash
make build-frontend            # writes docs/
git add docs/ && git commit -m "ops: publish frontend $(git describe)"
git push origin main           # Pages picks it up within ~60s
```

Roll back: `git revert <publish-commit>` and push.
