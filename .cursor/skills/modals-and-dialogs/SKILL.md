---
name: modals-and-dialogs
description: Use when building modals, sheets, drawers, or confirmation flows — Dialog vs Sheet vs AlertDialog, focus trapping, and destructive confirms.
---

# Modals & Dialogs

| Use | Primitive |
|---|---|
| Short focused task | Dialog |
| Inspector on narrow windows | Sheet |
| Irreversible | AlertDialog (no outside-click dismiss) |
| Command palette | CommandDialog |

- Always `DialogTitle` (sr-only if needed).
- Close on success only for forms.
- Warn if dirty.
- Do not stack modals.
- Prefer inline approvals in the transcript over a modal; modal only for checkout of loop/policy (L3).
