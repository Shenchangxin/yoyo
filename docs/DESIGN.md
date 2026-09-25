# Yoyo design system

Yoyo is a harness workstation: an operator watches an agent work, reviews what
it changed, and decides what survives. The interface should feel like a quiet
Mac utility — Xcode's inspector, Linear's density, Messages' composer, Final
Cut's viewer — not a chat product and not an IDE. Every rule below serves one
sentence:

> **Show the work, hide the chrome. Color is evidence. Motion is state.**

Visual direction is a warm night studio: Plus Jakarta Sans, terracotta on
charcoal, Apple HIG materials (glass chrome, concentric radii, grouped Form).
Beauty is instrument craft — not personality marketing. Skill-layer notes live
in `design-system/yoyo/MASTER.md`; this file still wins on grammar.

The source of truth for tokens is `frontend/src/styles.css`. This document
explains how to spend them. Do not follow stale Inter / sage / zinc recipes.

Hosted 影策 (`.yingce-island`) maps onto these tokens via
`frontend/src/features/video/canvas-host/design-system/` and a thin
`apple-skin.css`. Yingce is an engine, not a costume. Canvas chrome follows
**Studio Glass**: Liquid Glass materials with ProKit discipline. Glass is a
functional layer above opaque media. Do not mix Regular and Clear glass.

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
3. **Color is evidence.** Warm charcoal chrome. Terracotta (`accent`) is for
   primary selection and live generation. `success` and `danger` appear only
   where they carry meaning: diff adds/deletes, failed steps, destructive
   actions. `warning` marks incomplete state. Do not spend hue on node-type
   rainbows.
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
7. **One chrome.** PageHeader is the only toolbar. Video is Create | Drama |
   Canvas in that header. The rail on Video is projects, not a second product
   nav. Assets, plugins, and creation history jump from the command palette.
   Skills is the Yoyo Skills surface.

---

## 2. Tokens

### Color (semantic only — no raw hex in feature files)

Mode (`system | dark | light`) is orthogonal to named palettes. The document
element carries `class="dark|light"` and `data-palette`. Accent, success,
danger, and warning stay put so Harbor and diffs still mean the same thing.

Default dark is the warm night studio. Mist (cool daylight) is an opt-in light
palette — not the brand.

| Token | Dark (studio) | Light | Use |
|---|---|---|---|
| `background` | `#161310` | `#f5f5f3` | app canvas, transcript |
| `sidebar` | `#100e0c` | cool/warm rail | rail, review panel, code surfaces |
| `card` | `#1e1a17` | paper | actionable cards, grouped Form |
| `popover` | `#25201c` | white | menus, tooltips, jump pill |
| `input-bar` | `#1c1916` | white | composer |
| `lift` | `#2c2621` | hover wash | hover, selected, chips, kbd |
| `border` | foreground 10% | ink 10% | every hairline |
| `foreground` | `#f2ede6` | `#1c1916` | primary text, primary button fill |
| `muted` | `#9a9086` | `#6f675f` | secondary text, icons |
| `accent` | `#e08a6a` terracotta | terracotta | selection, live generation |
| `success` | `#7eae8c` | green | diff add, healthy, iOS switch |
| `warning` | `#c4a15a` | gold | interrupted, needs approval |
| `danger` | `#d16a64` | red | failed, destructive |
| `media-surface` | `#0b0d10` | near-black | Drama viewer, canvas stage |
| `glass-bg` | sidebar 72% | sidebar 72% | chrome only (header, rail, inspector) |

Opacity steps for text: `foreground` (primary) · `foreground/85` (labels) ·
`muted` (secondary) · `muted/70` (tertiary, meta) · `muted/50` (separators
like `·`). Do not invent new greys; step opacity.

Tinted surfaces use the `[0.11]` step for row backgrounds (`bg-success/[0.11]`)
and `/8`–`/12` for card fills. Never tint text and background of the same row
with the same hue — colour lives in **either** the marker **or** the fill.

### Typography

- **UI:** Plus Jakarta Sans Variable. Body 13px, labels 12.5px, meta 11.5px,
  eyebrow 10.5px uppercase `tracking-[0.08em]`. Headlines cap at **21px**/600/
  `tracking-[-0.03em]`. Pane titles 13px/500. Transcript prose is a three-step
  ramp: answer 13.5px/1.6, headings 16 / 14.5 / 13.5px, process 12.5px, code
  12px. Headings use `text-wrap: pretty`.
- **Code:** `--font-mono` (JetBrains Mono, then system mono). 11.5px/1.6 in
  diffs and tool payloads, 12px for tool names inline.
  `font-variant-ligatures: none` everywhere code is shown.
- **Numbers:** `tabular-nums` on every count, duration and size.

Do not use Inter. Do not use `fs-display` 36 / 72. Yingce island `--fs-display`
is remapped to 21px.

### Spacing & radius

4pt grid. Row heights: 28px (transcript process rows), 32px (list rows,
section labels), 36px (pane toolbars), 40px (tab strips), 44px (footers with a
primary action), 48px (PageHeader). Gutters: 24/32px transcript, 12px panels.

Concentric radii (Apple): 5px segmented inset, 6px icon buttons / kbd / chips,
8px (`--radius-control`) fields and default buttons, 10px inner panels,
12px nodes / grouped cards, 16px (`--radius-composer`) composer and docks,
18px user bubble, full pills and primary send.

### Elevation & glass

Depth comes from an inset top highlight (`--shadow-card`, `--shadow-composer`)
on cards and the composer, and a single soft drop shadow (`--shadow-popover`)
on floating layers. Glass (`backdrop-filter: blur(20px) saturate(160%)` on
`--glass-bg`) is for **chrome only**: PageHeader, SidebarCard, inspector,
canvas dock, menus. The stage and media surface stay opaque. Honour
`prefers-reduced-transparency` (solid `--sidebar` / `--popover`) and
`prefers-reduced-motion`.

The app canvas may carry a 2–5% film grain. No glows, no Spotlight blobs, no
HoverBorderGradient, no WorkingGlow.

### Mark

The four-cell CAS square is the only brand glyph. On empty still-lifes it sits
in a 40px inset well (`MarkWell`). Never replace it with Sparkles, a wordmark
illustration, or an avatar.

### Motion

| Token | Value | Use |
|---|---|---|
| `--duration-fast` | 140ms | hover, chevron rotate, row swap |
| `--duration` | 200ms | disclosure height, tab underline |
| `--duration-shell` | 240ms | pane resize, sheet |
| `--ease-out` | `cubic-bezier(.32,.72,0,1)` | everything |

Live grammar (see `ProcessGroup`): `.shimmer-text` sweeps the verb, `.pulse-dot`
breathes on the rail, `useNow` ticks a `Worked for` clock. All three stop the
moment `running` is false.

---

## 3. Layout

```
┌ rail ─┐┌──────────── stage ────────────┐┌── review ──┐
│ Yoyo  ││ PageHeader (drag region)      ││ tabs       │
│ New   ││ Create | Drama | Canvas       ││ toolbar    │
│ chats ││                               ││ list/diff  │
│ /proj ││   [ fluid column / studio ]   ││            │
│       ││   transcript … composer       ││ footer CTA │
│ Skills││   or canvas / drama split     ││            │
│ Video ││                               ││            │
│ Harn. │└───────────────────────────────┘└────────────┘
```

- App frame padding 8px; rail, review, Settings, Skills, and Harness are
  glass cards on `sidebar`; Agent and Video stages are bare `background`.
- **Video** is one chrome. Create / Drama / Canvas plus Assets / Skills /
  Plugins / History live in the left rail (`VideoShellNav`). Drama series and
  canvas boards live in Creation history, not a second workspace list under
  the rail. Yoyo Skills in the dock is the skill market; Video → Skills is
  the hosted island library. Studio image / video / speech adapters in
  Settings are the same providers Create and Canvas pick from.
- **Drama** is a Final Cut split: browser (cast/scenes/props) | viewer + film
  strip | inspector. Pipeline verbs sit on the viewer. Tasks open a drawer.
- **Canvas** keeps the Yingce renderer. Nodes are monochrome media tiles with
  a 1-letter type mark; terracotta only while generating. Chrome is grouped
  glass islands on the window edge plus a selection HUD. Agent pins to the
  right inspector — no second chat product. See **Canvas island**.
- **Harness process rail** — Overview is the unnumbered origin. Propose /
  Prove / Promote carry `01 02 03` and share the Review underline tab
  grammar.
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
  tokens; the process row (pulse + shimmer verb) is status. Copy/review
  actions sit at the foot and appear on hover once the turn is settled.
  Turns are separated by whitespace (40px), not rules.
- **Process** — one disclosure row per model round. Live: pulse dot + shimmer
  verb (`Reading`, `Running`, `Thinking`) + mono detail + clock. Settled:
  `Worked for 20s · Ran 1 command · Read 2 files` with a dot / ✕ / dashed-ring
  glyph for done / failed / interrupted. Expanded: a hairline step rail.
- **Artifacts** (patch, office file, MCP view) — card on `card/60` with icon
  tile, title, mono meta and pill actions (`Preview`, `Open in Review`).
- **Approval** — card with a `warning` left bar and icon tile, eyebrow, action
  title, the command in a code surface, then `Allow once` (primary) · `This
  chat` · `Always` · `Reject` with kbd hints `1 2 3 Esc`.
- **Error** — `danger` bar and tint for provider failures; neutral card for
  soft stops with a `Continue this turn` CTA when retryable.
- **Empty thread** — bottom-anchored above the composer: mark, greeting
  (`Ready when you are.` / Video: `Paste a chapter.`), one hint line, starter
  **pills** (not a 3-column marketing grid). No headline larger than 21px.
- **Composer** — brightest object on the stage. Toolbar is ghost controls;
  send is a filled circle that flips to a stop square while running.

---

## 5. Review panel grammar

- **Tabs** — text tabs with a 1.5px underline; counts as small tabular
  numerals after the label. `aria-label` carries the label at every width.
- **Toolbar** (36px) — left: a fact (`2 files · 2 hunks`); right: icon
  segmented control and icon buttons. No labelled buttons in toolbars.
- **Diff** — grouped by file. Sticky file header with a tri-state checkbox,
  path (directory dimmed, basename strong, truncated from the left), hunk
  count. Footer appears only with a selection: `Apply n` · `Clear`.
- **Files / Queue / Memory** — `divide-y` hairline lists under uppercase
  eyebrows with counts.
- **Empty** — centred icon tile, one sentence, one hint.

---

## 6. Components

| Piece | Recipe |
|---|---|
| Icon button | `size-6 rounded-md text-muted hover:bg-lift hover:text-foreground`, always with `aria-label` and a tooltip |
| Text action | `h-6 px-2 rounded-md text-[11px] font-medium`; primary = `bg-foreground text-background`; danger only on hover |
| Segmented | `bg-lift/80 p-0.5 rounded-[10px]`; active segment `bg-card` 8px radius (iOS) |
| Code surface | `rounded-lg border-border/60 bg-sidebar/70 px-3 py-2 font-mono text-[11px] leading-[1.55]` |
| Section eyebrow | `h-8 px-3 text-[10.5px] font-medium uppercase tracking-[0.08em] text-muted/80` + tabular count |
| Grouped Form | macOS Settings: 12px card, 13px labels, trailing controls, hairlines between rows |
| kbd | 20px tall, `lift` fill, `border`, 10px/500 |
| Pill CTA | `rounded-full px-3 py-1.5 text-[12px] font-medium`; filled `foreground` for primary |
| Canvas node | 12px radius, hairline, monochrome type glyph, terracotta outline while generating or selected |
| Canvas island | grouped glass toolbar, 16px shell / 8px items, no magnification, icon-only |
| Overlay | Island · Menu · Popover · Dialog · Workbench. No new raw `antd.Modal`. |

---

## 7. Copy voice

Short, declarative, present tense for live (`Reading …`), past tense for
settled (`Read 2 files`). Sentences end with a period; labels do not. Prefer
the operator's vocabulary (turn, hunk, harness, workspace) over product
marketing. Avoid “loading”, “please”, exclamation marks and emoji. Chinese copy
mirrors the same structure — no 请 / 您 / extra particles. Video empty: “Paste
a chapter.” or “Drop a still.” One hint. One action.

---

## 8. Accessibility

Visible focus rings (`--ring`) on all controls; a skip link to `#main-stage`;
`aria-expanded` on every disclosure; `aria-live="polite"` on the single live
status line per turn; `role="tab"`/`tablist` with stable `aria-label`s; kbd
hints are `aria-hidden` so button names stay exact. Contrast: body text ≥ 7:1
on `background`, muted ≥ 4.5:1. `prefers-reduced-motion` and
`prefers-reduced-transparency` are honoured.

---

## 9. Forbidden

Avatars in the transcript · spinners on text · cards inside cards · bordered
list items · labelled buttons in toolbars · colour on chrome · ligatures in
code · headlines over 21px · Sparkles / WandSparkles · gradients · glows ·
HoverBorderGradient · SpotlightSurface · WorkingGlow · daisyUI · marketing
empty states · fake sandbox badges · a second agent loop in the client · a
7-item Video shell · Inter / sage / zinc as the live system · Apple-blue + SF
Pro as the default brand · credit-cost chips in the hosted island · a Chat
dock on Video · Dock magnification · SpotlightSurface on menus · purple agent
orbs · card-in-card create grids.

---

## 10. Canvas island (Studio Glass)

Hosted infinite canvas (`.yingce-island`) is a media stage, not a second
product. Tokens live in
`frontend/src/features/video/canvas-host/design-system/tokens.css`.

### Chrome

- Top bar 44px, transparent. Three glass islands: project (back/title/sync),
  tools (select/hand, undo/redo, add), trailing (search/version/share/focus).
- No bottom function dock. Bottom-left is only a zoom capsule (`− 100% +`).
- View extras (minimap, arrange, hide wires, shortcuts) live in a menu on the
  zoom capsule, not a second dock.
- Selection HUD attaches above the node. Primary actions ≤ 4, then More.
  Multi-select HUD: storyboard · send to Agent · More (alignment inside More).
- Agent is a right inspector. Launcher is a 32px ghost, never a purple orb.

### Surfaces

1. **Island** — toolbars, zoom, HUD. Glass, no title, no dimming layer.
2. **Menu** — context, overflow, create. 224–280px, row 32px, 16px glyphs, kbd
   on the right. No icon wells, no spotlight, no gradient rules.
3. **Popover** — generation settings and pickers. Anchored to the trigger.
4. **Dialog** — confirms and small settings. Opaque `popover`, 14px radius.
5. **Workbench** — timeline, style center, image edit family, drawing. Opaque.
   Full-height sheets are never glass.

Add morphs from the top-bar `+` into the create **Menu** (one list, grouped by
node / resource / project). Same menu is used from the canvas context menu.

### Node tile

Media full-bleed. Caption lives in the 22px external header. Selection is a
2px accent ring with 2px offset — not a drop shadow. Generating uses the same
ring. One badge at a time (error > lock > batch). Visible 6px corner handles
when selected.

### Mutex

At most one Menu, one Popover, and one HUD. Opening a Dialog or Workbench
dismisses HUD popovers. Chrome never raises itself above the node HUD.
