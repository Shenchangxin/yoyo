---
name: web-design-guidelines
description: Review UI code for Web Interface Guidelines compliance. Use when asked to review UI, check accessibility, audit design, review UX, or check the client against best practices.
---

# Web Interface Guidelines

Review frontend files against Vercel Web Interface Guidelines.

## How it works

1. Prefer fetching latest rules from `https://raw.githubusercontent.com/vercel-labs/web-interface-guidelines/main/command.md`.
2. If offline, use the checklist below (snapshot of the public guidelines).
3. Output `file:line` findings, grouped by file. Terse. No preamble.

## Checklist (Yoyo-relevant)

### Accessibility

- Icon-only buttons need `aria-label`
- Form controls need a label or `aria-label`
- `button` for actions; `a` for navigation
- Decorative icons `aria-hidden`
- Async updates: `aria-live="polite"`
- Semantic landmarks (`nav`, `main`, `aside`)
- Headings hierarchical

### Focus

- Visible `focus-visible:ring-*`
- Never `outline-none` without a replacement
- Sticky chrome must not cover focused elements

### Forms

- Correct `type`, `name`, `autoComplete`
- Labels clickable (`htmlFor`)
- Inline errors; submit disabled only after request starts
- Placeholders end with `…`

### Animation

- Honor `prefers-reduced-motion`
- Animate transform/opacity only
- Never `transition: all`

### Typography / content

- `…` not `...`
- Tabular nums for number columns
- `text-wrap: pretty` on headings
- Truncate with `min-w-0` on flex children

### Performance

- Lists >50 items: virtualize
- Controlled inputs must be cheap per keystroke
- `color-scheme: dark` on `html` for dark apps

### Navigation & state

- Deep-link tabs/panels where a desktop client can (query or persisted layout)
- Destructive actions: confirm or undo

### Anti-patterns

- Icon buttons without `aria-label`
- Large `.map()` without virtualization
- Hand-rolled modal without focus trap
- Hardcoded date formats (use `Intl`)
