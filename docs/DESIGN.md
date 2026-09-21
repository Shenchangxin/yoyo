# Yoyo design system

Yoyo is a harness workstation: an operator watches an agent work, reviews what
it changed, and decides what survives. The interface should feel like a quiet
Mac utility — Xcode's inspector, Linear's density, Codex's transcript grammar —
not a chat product and not an IDE. Every rule below serves one sentence:

> **Show the work, hide the chrome. Color is evidence. Motion is state.**

Visual direction is Swiss Modernism 2.0 applied to a Mac utility: Inter, a
4pt grid, one sage accent, mathematical spacing. Beauty is instrument craft
(CAS mark, pointer maps, paper grain) — not personality marketing. Skill-layer
notes live in `design-system/yoyo/MASTER.md`; this file still wins on grammar.

The source of truth for tokens is `frontend/src/styles.css`. This document
explains how to spend them.

---

## 1. Principles

1. **One conversation column.** Prose, bubbles, process rows and the composer
   share a single centred measure (`THREAD_COL`). It tracks the stage width
   (`min(100%, var(--thread-measure))`, ceiling 72rem) so the session is
   responsive; left edges still align. The scroll container stays pane-wide
   so the scrollbar hugs the pane edge.
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

Mode (`system | dark | light`) is orthogonal to named palettes. The document
element carries `class="dark|light"` and `data-palette`. Accent, success,
danger, and warning stay put so Harbor and diffs still mean the same thing.

Dark palettes: **Ink** (default, `#0c0d0e`), Dim (lifted charcoal), Slate
(cool blue-black). Light palettes: **Neutral** (default, true gray `#f4f4f5`),
Paper (warm ivory `#f1f0ea`, opt-in), Mist (cool daylight). Do not ship a
yellow light as the only option.

| Token | Dark (Ink) | Light (Neutral) | Use |
|---|---|---|---|
| `background` | `#0c0d0e` | `#f4f4f5` | app canvas, transcript |
| `sidebar` | `#141516` | `#e8e8ea` | rail, review panel, code surfaces |
| `card` | `#1a1b1c` | `#ffffff` | actionable cards |
| `popover` | `#222324` | `#ffffff` | menus, tooltips, jump pill |
| `input-bar` | `#1c1d1e` | `#ffffff` | composer |
| `lift` | `#262728` | `#e4e4e7` | hover, selected, chips, kbd |
| `border` | white 9% | black 8.5% | every hairline |
| `foreground` | `#ececea` | `#18181b` | primary text, primary button fill |
| `muted` | `#8c8e8d` | `#5f5f66` | secondary text, icons |
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
  titles). Transcript prose is a three-step ramp, not one size: answer 13.5px/1.6,
  headings 16 / 14.5 / 13.5px, process 12.5px, code 12px. Headings use
  `text-wrap: pretty` so widows do not sit alone.
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
(`--shadow-popover`) on floating layers only. The app canvas may carry a 2–5%
film grain (material, not a glow). No glows, no blur beyond `backdrop-blur-sm`
on sticky headers.

### Mark

The four-cell CAS square is the only brand glyph. On empty still-lifes it sits
in a 40px inset well (`MarkWell`). Never replace it with Sparkles, a wordmark
illustration, or an avatar.

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
│ chats ││   [ fluid column ≤72rem ]     ││ list/diff  │
│       ││   transcript … composer       ││            │
│ Skills││                               ││ footer CTA │
│ Harn. │└───────────────────────────────┘└────────────┘
```

- App frame padding 8px; rail, review, Settings, Skills, and Harness are
  rounded cards on `sidebar` with the inset highlight; the agent stage is
  bare `background` so the transcript reads as a document.
- **Harness process rail** — Overview is the unnumbered origin. Propose /
  Prove / Promote carry `01 02 03` and share the Review underline tab
  grammar. They are one RSI sequence, not four sibling labs.
- Default split 19 / 51 / 30. Review collapses to a right sheet under 1100px;
  the rail becomes a hover overlay under 800px. Panes are user-resizable and
  persisted (`yoyo-layout-v1`).
- The review panel is a `@container`; tab labels show at ≥320px, icons below.

---

## 4. Transcript grammar

- **User** — right-aligned bubble on `lift`, 18px radius, ≤80% of the column,
  13px. No avatar.
- **Assistant** — left prose, no bubble, no avatar. One turn is one
  `assistant-letter`. The answer is the scan target (13.5px); headings step
  16 / 14.5 / 13.5px — never Streamdown's stock `text-3xl`. Process rows and
  fenced code sit a step quieter (12.5px / 12px). A circle caret marks live
  tokens; the process row (pulse + shimmer verb) is status. The letter stays
  at answer contrast so it remains readable while it grows. Copy/review
  actions sit at the foot and appear on hover once the turn is settled.
  Turns are separated by whitespace (40px), not rules.
- **Process** — one disclosure row per model round. Live: pulse dot + shimmer
  verb (`Reading`, `Running`, `Thinking`) + mono detail + clock. Settled:
  `Worked for 20s · Ran 1 command · Read 2 files` with a dot / ✕ / dashed-ring
  glyph for done / failed / interrupted. Expanded: a hairline step rail; each
  row is glyph · mono tool name · mono detail · elapsed · state, with input and
  output code surfaces beneath. A call without a result is **interrupted** the
  moment the turn ends — never a spinner on a finished turn.
- **Artifacts** (patch, office file, MCP view) — card on `card/60` with icon
  tile, title, mono meta and pill actions (`Preview`, `Open in Review`).
  Host-open is Review's job, not a second button on the card. Pending shows
  shimmer on the title.
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
  small text actions (`Once` primary, `Deny` danger-on-hover). Files preview
  fills the remaining Review column; source is Shiki-highlighted, HTML is
  sandboxed, binary is a one-line note.
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

Visible focus rings (`--ring`) on all controls; a skip link to `#main-stage`;
`aria-expanded` on every
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
