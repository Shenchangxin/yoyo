---
name: navigation-patterns
description: Use when building app navigation — sidebar nav with active states, a command palette (cmdk), tabs, and breadcrumbs. Covers accessible markup and keyboard-first navigation.
---

# Navigation Patterns

Yoyo primary nav is **threads**, not SaaS page links. Labs are secondary and must not look like five equal destinations.

- Thread list: `aria-current="page"` on the active thread. Running state is text + a dot, not color-only.
- Labs: a compact footer or a single "Labs" menu, not a second primary IA.
- Command palette: shadcn `Command` / cmdk. Groups: Threads, Labs, Actions. Fuzzy search. Esc closes.
- Inspector tabs: Radix Tabs; persist last tab per session.
- Native File menu already creates sessions — keep in sync with the rail.

Do not implement Next.js `Link`/`usePathname`. This is a Wails single-page shell.
