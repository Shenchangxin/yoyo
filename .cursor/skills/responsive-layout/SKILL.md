---
name: responsive-layout
description: Use when building the app shell or any responsive screen — sidebar + topbar layouts, Tailwind breakpoints, container queries, and fluid grids that hold up from mobile to ultrawide.
---

# Responsive Layout

Yoyo is a **desktop workstation**. Primary shell is three panes, not a marketing `min-h-screen` site.

```tsx
<div className="flex h-full min-h-0 bg-background">
  <aside className="hidden w-[260px] shrink-0 border-r md:flex">…threads…</aside>
  <main className="flex min-w-0 min-h-0 flex-1 flex-col">{children}</main>
  <aside className="hidden w-[360px] shrink-0 border-l xl:flex">…review…</aside>
</div>
```

- `h-full min-h-0` not `h-screen` inside Wails webview.
- `min-w-0` on every flex child that must truncate.
- Below ~1100px: collapse inspector to a Sheet; below ~800px: thread rail becomes a Sheet.
- Pane widths: `react-resizable-panels`, persist to localStorage.
- Overlay titlebar: `--wails-draggable` on the top 36px; interactive controls `no-drag`.

## Pitfalls

- CSS grid with fixed `260px / 1fr / 360px` and no collapse.
- Horizontal overflow from diffs/code because of missing `min-w-0`.
- Viewport queries for components reused at different widths — use `@container`.
