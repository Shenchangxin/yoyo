---
name: accessible-components
description: Use when building or reviewing interactive UI that must meet WCAG 2.2 AA — focus management, ARIA roles and states, keyboard navigation, labels, and visible focus.
---

# Accessible Components

Use real elements and Radix/shadcn primitives. Do not reinvent dropdowns, palettes, or dialogs with a `div`.

- `button` not clickable `div`.
- Every input has a label (`htmlFor` or wrap).
- Icon-only buttons: `aria-label`. Decorative icons: `aria-hidden`.
- Visible focus: `focus-visible:ring-2 focus-visible:ring-ring`.
- `Esc` closes overlays; focus returns to the trigger.
- Live regions for "Working…", toasts, approval banners (`role="status"`).
- Contrast ≥4.5:1. State is never color-only.

Yoyo-specific: command palette must be a focus-trapped dialog (cmdk). Mention menu must be a listbox with arrow keys. Approval actions must be reachable without the inspector open.
