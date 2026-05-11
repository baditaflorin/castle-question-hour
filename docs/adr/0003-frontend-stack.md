# 0003 — Frontend framework & build tooling

- Status: accepted
- Date: 2026-05-11

## Decision

- **Vite** as the build tool. Fast HMR, native ESM, first-class TS, painless static output.
- **React 18** as the rendering layer. Small surface needed (one-question screen, one-summary
  screen, one-settings screen) — React is overkill in absolute terms but ergonomic and well-known
  enough to keep contribution barriers low.
- **TypeScript strict** mode.
- **Tailwind CSS** for styling. No component library — three screens don't need one.
- **Yjs** + **y-webrtc** + **y-indexeddb** for the CRDT data plane.
- **TanStack Query** for backend fetches (current question, summary trigger).
- **Zod** for runtime payload validation at the boundary.

## Consequences

- The build is a static `docs/` folder with hashed asset filenames, base path `/castle-question-hour/`.
- Service worker registered for Web Push and offline question replay.
- Initial JS payload under 200KB gz before lazy chunks.

## Alternatives considered

- SvelteKit: produces a perfectly good static site, but the team's React fluency wins.
- Astro: islands architecture is wasted here — the whole app is one interactive surface.
- Vanilla TS: tempting but Yjs awareness + multiple synchronized views benefit from React's
  re-render model.
