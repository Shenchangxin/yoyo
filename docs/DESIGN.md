# Yoyo design system

Yoyo is a harness workstation: an operator watches an agent work, reviews what
it changed, and decides what survives. The interface should feel like a quiet
Mac utility — Xcode's inspector, Linear's density, Codex's transcript grammar —
not a chat product and not an IDE. Every rule below serves one sentence:

> **Show the work, hide the chrome. Color is evidence. Motion is state.**

The source of truth for tokens is `frontend/src/styles.css`. This document
explains how to spend them.

---

## 1. Principles

1. **One reading column.** Prose, bubbles, process rows and the composer share
   a single centred measure (`THREAD_COL`, 46rem). Left edges align. The scroll
   container stays pane-wide so the scrollbar hugs the pane edge.
2. **Hairlines, not boxes.** Lists are separated by `border-border/50` lines.
   Cards are reserved for objects the operator acts on (approvals, artifacts,
   errors). Never nest a card inside a card.
3. **Color is evidence.** Zinc chrome. `success` and `danger` appear only where
   they carry meaning: diff adds/deletes, failed steps, destructive actions.
   `warning` marks incomplete state (interrupted, needs approval). `accent`
   is for primary selection in the composer and Harbor verdicts.
4. **Motion is state.** Something moves only while something is happening
   (shimmer verb, pulse dot, ticking clock). The instant work ends the motion
   stops and the row settles into past tense. Decorative motion is limited to
   enter transitions ≤ 320ms.
5. **Say less, in order.** Headline → one-line context → action. Empty states
   are one sentence plus one hint. Buttons are verbs. Counts are tabular
   numerals, never in parentheses.
6. **Truth in the audit trail.** Narration (“Read 3 files”) lives in summaries;
   raw tool names, paths and commands stay visible in expanded rows and diffs
   in monospace with ligatures off.

---

## 2. Tokens

### Color (semantic only — no raw hex in feature files)

| Token | Dark | Light | Use |
|---|---|---|---|
| `background` | `#0f1011` | `#f3f3f0` | app canvas, transcript |
| `sidebar` | `#161718` | `#e9e9e4` | rail, review panel, code surfaces |
| `card` | `#1a1b1c` | `#fafaf8` | actionable cards |
| `popover` | `#1f2021` | `#ffffff` | menus, tooltips, jump pill |
| `input-bar` | `#1c1d1e` | `#ffffff` | composer |
| `lift` | `#252627` | `#e2e2dc` | hover, selected, chips, kbd |
| `border` | white 9% | black 8.5% | every hairline |
| `foreground` | `#ececea` | `#1b1b1b` | primary text, primary button fill |
| `muted` | `#8c8e8d` | `#6b6b66` | secondary text, icons |
| `accent` | `#6d8a7e` | `#5d7368` | primary selection, Harbor pass |
| `success` | `#6fae82` | `#3f8a58` | diff add, healthy |
| `warning` | `#d1a24a` | `#a97a1f` | interrupted, needs approval |
| `danger` | `#dc3d3d` | `#c43333` | failed, destructive |

Opacity steps for text: `foreground` (primary) · `foreground/85` (labels) ·
`muted` (secondary) · `muted/70` (tertiary, meta) · `muted/50` (separators
like `·`). Do not invent new greys; step opacity.

Tinted surfaces use the `[0.11]` step for row backgrounds (`bg-success/[0.11]`)
and `/8`–`/12` for card fills. Never tint text and background of the same row
with the same hue — colour lives in **either** the marker **or** the fill.

### Typography

- **UI:** Inter Variable, `cv11 ss03 calt`, optical sizing on. Body 13px,
  labels 12.5px, meta 11.5px, eyebrow 10.5px uppercase `tracking-[0.08em]`.
  Headlines: 21px/600/`tracking-[-0.03em]` (empty state), 13px/500 (pane
  titles). Prose 15.5px/1.7 with `-0.011em` tracking, max 72ch.
- **Code:** `--font-mono` (system mono ladder). 11.5px/1.6 in diffs and tool
  payloads, 12px for tool names inline. `font-variant-ligatures: none`
  everywhere code is shown — a diff must show the bytes on disk.
- **Numbers:** `tabular-nums` on every count, duration and size so columns do
  not jitter while live.

### Spacing & radius

4pt grid. Row heights: 28px (transcript process rows), 32px (list rows,
section labels), 36px (pane toolbars), 40px (tab strips, pane headers), 44px
(footers with a primary action). Gutters: 24/32px transcript, 12px panels.

Radius: 5px (segmented buttons), 6px (icon buttons, kbd, chips), 8px (code
surfaces), 10px (`--radius`, panes and cards), 16px (`--radius-composer`),
18px (user bubble), full (pills, jump-to-latest, primary CTAs).

### Elevation

Flat. Depth comes from an inset top highlight (`--shadow-card`,
`--shadow-composer`) on cards and the composer, and a single soft drop shadow
(`--shadow-popover`) on floating layers only. No glows, no blur beyond
`backdrop-blur-sm` on sticky headers.

### Motion

| Token | Value | Use |
|---|---|---|
| `--duration-fast` | 120ms | hover, chevron rotate, row swap |
| `--duration` | 180ms | disclosure height, tab underline |
| `--duration-shell` | 240ms | pane resize, sheet |
| `--ease-out` | `cubic-bezier(.22,.61,.36,1)` | everything |

Live grammar (see `ProcessGroup`): `.shimmer-text` sweeps the verb, `.pulse-dot`
breathes on the rail, `useNow` ticks a `Worked for` clock. All three stop the
moment `running` is false. `prefers-reduced-motion` freezes shimmer to plain
foreground text and removes pulse, rise and caret animations.

---

## 3. Layout

```
┌ rail ─┐┌──────────── stage ────────────┐┌── review ──┐
│ Yoyo  ││ PageHeader (drag region)      ││ tabs       │
│ New   ││                               ││ toolbar    │
│ chats ││   [ reading column 46rem ]    ││ list/diff  │
│       ││   transcript … composer       ││            │
│ Skills││                               ││ footer CTA │
│ Harn. │└───────────────────────────────┘└────────────┘
```

- App frame padding 8px; rail and review are rounded cards on `sidebar` with
  the inset highlight; the stage is bare `background`.
- Default split 19 / 51 / 30. Review collapses to a right sheet under 1100px;
  the rail becomes a hover overlay under 800px. Panes are user-resizable and
  persisted (`yoyo-layout-v1`).
- The review panel is a `@container`; tab labels show at ≥320px, icons below.

---

## 4. Transcript grammar

- **User** — right-aligned bubble on `lift`, 18px radius, ≤34rem. No avatar.
- **Assistant** — left prose, no bubble, no avatar. One turn is one
  `assistant-letter`; copy/review actions sit at the foot and appear on hover
  once the turn is settled. Turns are separated by whitespace (40px), not rules.
- **Process** — one disclosure row per model round. Live: pulse dot + shimmer
  verb (`Reading`, `Running`, `Thinking`) + mono detail + clock. Settled:
  `Worked for 20s · Ran 1 command · Read 2 files` with a dot / ✕ / dashed-ring
  glyph for done / failed / interrupted. Expanded: a hairline step rail; each
  row is glyph · mono tool name · mono detail · elapsed · state, with input and
  output code surfaces beneath. A call without a result is **interrupted** the
  moment the turn ends — never a spinner on a finished turn.
- **Artifacts** (patch, office file, MCP view) — card on `card/60` with icon
  tile, title, mono meta and pill actions. Pending shows shimmer on the title.
- **Approval** — card with a `warning` left bar and icon tile, eyebrow, action
  title, the command in a code surface, then `Allow once` (primary) · `This
  chat` · `Always` · `Reject` with kbd hints `1 2 3 Esc`.
- **Error** — `danger` bar and tint for provider failures; neutral card for
  soft stops (`Stopped`, `Reached the turn limit`) with a `Continue this turn`
  CTA when retryable. Details behind a disclosure.
- **Empty thread** — bottom-anchored above the composer: mark, `Ready when you
  are.`, one hint line (`Working in ws · ↵ sends · @ pins context · ⇧⇥
  toggles Plan`), three starter pills. No headline larger than 21px, no
  marketing copy.
- **Composer** — brightest object on the stage. Toolbar is ghost controls;
  send is a filled circle that flips to a stop square while running. The
  context meter appears only once tokens exist or a turn is running or queued.

---

## 5. Review panel grammar

- **Tabs** — text tabs with a 1.5px underline; counts as small tabular
  numerals after the label. `aria-label` carries the label at every width.
- **Toolbar** (36px) — left: a fact (`2 files · 2 hunks`); right: icon
  segmented control and icon buttons. No labelled buttons in toolbars.
- **Diff** — grouped by file. Sticky file header with a tri-state checkbox,
  path (directory dimmed, basename strong, truncated from the left), hunk
  count. Each hunk: checkbox + `@@` header row, then tinted diff rows with a
  14px marker gutter. Footer appears only with a selection: `Apply n` · `Clear`.
- **Files / Queue / Memory** — `divide-y` hairline lists under uppercase
  eyebrows with counts. Row actions are icon buttons revealed on hover or
  small text actions (`Once` primary, `Deny` danger-on-hover).
- **Trace** — one stats line (`10 events · 3 tools · 0 errors · 2.1s · 12k
  tokens`), filter chips, then rows: time · mono type · summary · elapsed.
  Expanded rows show a code surface and a Copy action.
- **Empty** — centred icon tile, one sentence, one hint. Never instructions
  longer than a line.

---

## 6. Components

| Piece | Recipe |
|---|---|
| Icon button | `size-6 rounded-md text-muted hover:bg-lift hover:text-foreground`, always with `aria-label` and a tooltip |
| Text action | `h-6 px-2 rounded-md text-[11px] font-medium`; primary = `bg-foreground text-background`; danger only on hover |
| Segmented | `bg-lift/70 p-0.5 rounded-md`; active segment `bg-panel shadow-[var(--shadow-card)]` |
| Code surface | `rounded-lg border-border/60 bg-sidebar/70 px-3 py-2 font-mono text-[11px] leading-[1.55]`, `max-h` capped, scrolls |
| Section eyebrow | `h-8 px-3 text-[10.5px] font-medium uppercase tracking-[0.08em] text-muted/80` + tabular count |
| kbd | 20px tall, `lift` fill, `border`, 10px/500 |
| Pill CTA | `rounded-full px-3 py-1.5 text-[12px] font-medium`; filled `foreground` for primary, `border-border` for secondary |

---

## 7. Copy voice

Short, declarative, present tense for live (`Reading …`), past tense for
settled (`Read 2 files`). Sentences end with a period; labels do not. Prefer
the operator's vocabulary (turn, hunk, harness, workspace) over product
marketing. Avoid “loading”, “please”, exclamation marks and emoji. Chinese copy
mirrors the same structure — no added politeness particles.

---

## 8. Accessibility

Visible focus rings (`--ring`) on all controls; `aria-expanded` on every
disclosure; `aria-live="polite"` on the single live status line per turn;
`role="tab"`/`tablist` with stable `aria-label`s; kbd hints are `aria-hidden`
so button names stay exact. Contrast: body text ≥ 7:1 on `background`, muted ≥
4.5:1. Reduced motion is honoured for every animation in this document.

---

## 9. Forbidden

Avatars in the transcript · spinners on text · cards inside cards · bordered
list items · labelled buttons in toolbars · colour on chrome · ligatures in
code · headlines over 21px · “Sparkles”, gradients, glows · daisyUI · marketing
empty states · fake sandbox badges · a second agent loop in the client.
