---
name: design-tokens-theming
description: Use when setting up a design system — color, typography, and spacing scales as CSS variables, the shadcn/ui theme, and dark mode. Covers semantic tokens, Tailwind wiring, and consistent theming across components.
---

# Design Tokens & Theming

Name tokens by **role**, not hue. `--primary`, `--muted`, `--destructive` — not `--blue-500`.

Yoyo is a dark-first desktop client. Put semantic channels on `:root` (and `html { color-scheme: dark }`), expose them through Tailwind v4 `@theme`, and never sprinkle raw hex in feature files.

```css
@import "tailwindcss";
:root {
  --background: 0 0% 5%;
  --foreground: 0 0% 93%;
  --sidebar: 0 0% 9%;
  --panel: 0 0% 13%;
  --muted: 0 0% 56%;
  --primary: 160 82% 35%; /* workstation teal, not decoration */
  --destructive: 0 72% 51%;
  --border: 0 0% 18%;
  --radius: 0.75rem;
}
html { color-scheme: dark; }
```

## Scales

- Spacing: Tailwind 4px scale only. No `p-[13px]`.
- Type: 6 steps (`text-xs` … `text-2xl`). Product UI does not need fluid display type.
- Radius: one `--radius`; pills for small chips; composer may use a larger token (`--radius-composer`) — name it, do not invent `rounded-[28px]` ad hoc.
- Elevation: border **or** shadow, not both (ghost card).

## Pitfalls

- Raw hex in components (`#10a37f`, `#0d0d0d`) — move to tokens.
- `dark:` overrides instead of flipping tokens.
- Too many accent colors. One action accent + semantic danger/success.
