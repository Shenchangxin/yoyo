---
name: react-best-practices
description: React performance guidelines from Vercel Engineering. Use when writing, reviewing, or refactoring React components, data fetching, bundle size, or render performance in the Yoyo desktop client.
---

# React Best Practices

Client-first subset for Wails + Vite (no RSC/Next). Full Vercel list is 70 rules; apply these in this repo.

## Critical

- Independent fetches in `Promise.all` (already used in `App.refresh`; do not add sequential awaits after it without need).
- Import lucide icons by name, never `import * as Icons`.
- Dynamic-import heavy panes (labs, markdown, future diff editor) so the agent shell stays small.
- Do not introduce barrel `index.ts` files that re-export the world.

## Re-renders (this codebase's actual problem)

- `App.tsx` is a god store (~30 `useState`s). Split into `ShellStore` (zustand/jotai) so composer keystrokes do not rerender the thread rail and inspector.
- Do not define components inside components (`ItemRow` is fine at module scope).
- Transcript: subscribe to the active session list, not a cloned `acc.slice()` of every item on every token if a store selector can isolate the last assistant message.
- Composer: keep the textarea's value local or in a store slice; do not lift every keystroke through a parent that also owns labs.
- `useDeferredValue` / `startTransition` for thread search filtering.
- `content-visibility: auto` or `virtua` once a transcript exceeds ~50 items.

## Client fetching

- Event-driven updates already exist (`yoyo:item`, SSE). Do not poll except the 1.5s running safety net.
- Deduplicate `approvals()` / `contextUsage()` in-flight.

## Rendering

- Prefer ternary over `&&` for possibly-zero counts.
- Theme native `<select>` / scrollbars; `color-scheme: dark` on `html`.
