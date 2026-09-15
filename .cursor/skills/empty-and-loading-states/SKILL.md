---
name: empty-and-loading-states
description: Use when handling non-happy UI states — skeletons, empty states with a CTA, error states with retry, and optimistic updates.
---

# Empty & Loading States

Every fetched view: loading, empty, error, success.

- Skeletons that match layout (`aria-busy`). No full-window spinner.
- First-use empty ≠ search-empty. The agent empty state must teach `@file` / Plan / workspace setup, not "What can I help with?" Sparkles.
- Error + retry. Banner errors in `App` must be dismissible **and** offer Open Control / Retry.
- Optimistic thread rename/fork; rollback + toast on failure.
