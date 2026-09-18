# Yoyo design

Operate-mode workstation. Steal Codex density, Work Buddy settings authority, and Vetta shell craft. Do not clone any of them.

## World

- **Dark-first** zinc surface. Light mode exists via the same semantic tokens.
- **One teal accent** for primary send and selection (`--accent`).
- Inter Variable for UI. System monospace for diffs, hashes, and tool payloads.
- Radius scale: controls `--radius`, composer `--radius-composer`, shell cards `10px`.
- Extra tokens: `--card`, `--popover`, shared ease `cubic-bezier(0.22, 0.61, 0.36, 1)`.

## Tokens only

Features use `bg-background`, `bg-panel`, `bg-sidebar`, `bg-card`, `bg-lift`, `text-foreground`, `text-muted`, `border-border`, `text-accent`, `text-danger`. No raw hex in feature files. Theme lives in `frontend/src/styles.css` (`html.dark` / `html.light`) plus `color-scheme`.

## Layout

Padded app frame (`p-2`). Sidebar is a rounded card: brand + active hash, New chat, search, threads, one Harness row, notifications + settings gear. Main column has a drag `PageHeader`. Three resizable panes on Agent. Review collapses to a sheet under 1100px. Sidebar becomes an overlay under 800px. Harness is one RSI workspace (Overview / Propose / Prove / Promote); chat docks as an optional right rail, off by default. Settings occupies the main column and hides the thread rail (200px nav + 680px content).

## Transcript and composer

- Transcript uses the main pane (gutters only). Composer stays a centered 48rem column. User bubbles right; assistant prose left, max ~75ch.
- Process (tools, reasoning, injections) folds per model round: live shows the in-flight step, settled collapses to “N tools”. A turn is one letter — one copy for all assistant prose, process lines are not selectable. Outcomes stay open: assistant prose, patch/artifact cards, approvals. Streaming caret only while tokens arrive. Stick-to-bottom yields when the user scrolls up.
- Enter sends. Shift+Enter newline. Running turns expose Steer next to Stop. Composer disabled with an inline reason until a workspace exists.
- Empty state: three operator prompts + workspace CTA. No Sparkles headline.

## Review

Split or unified highlighter. File list from the git diff. Apply selected with an undo toast. Context meter uses tabular nums and layer chips (trusted pin vs untrusted tools). Last inspector tab is remembered per thread.

## Motion and a11y

`prefers-reduced-motion` disables caret blink, section breathe, and decorative transitions. Visible focus rings on controls. Keyboard: New chat, send, stop, palette, Once/Deny, toggle review, settings (`Mod+,`). On Harness, `Mod+1`–`Mod+4` switch Overview / Propose / Prove / Promote. Cmd+K indexes settings sections and deep-links with a breathe highlight.

## Forbidden

daisyUI, Next.js, Monaco as the home view, marketing empty states, fake “sandboxed” chrome.
