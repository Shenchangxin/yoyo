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

Padded app frame (`p-2`). Sidebar is a rounded card: brand, compact lab nav, threads, notifications + settings gear. Main column is a separate rounded card with a drag `PageHeader`. Three resizable panes inside the frame. Review collapses to a sheet under 1100px. Sidebar becomes an overlay under 800px. Labs keep the thread rail; chat docks as a right rail. Settings is a gear surface occupying the main column (200px nav + 680px content), not a fifth lab.

## Transcript and composer

- User bubbles right; assistant prose left, max ~75ch.
- Tools as compact disclosure rows. Streaming caret. Stick-to-bottom yields when the user scrolls up.
- Enter sends. Shift+Enter newline. Composer disabled with an inline reason until a workspace exists.
- Empty state: three operator prompts + workspace CTA. No Sparkles headline.

## Review

Split or unified highlighter. File list from the git diff. Apply selected with an undo toast. Context meter uses tabular nums and layer chips (trusted pin vs untrusted tools). Last inspector tab is remembered per thread.

## Motion and a11y

`prefers-reduced-motion` disables caret blink, section breathe, and decorative transitions. Visible focus rings on controls. Keyboard: New chat, send, stop, palette, Once/Deny, toggle review, settings (`Mod+,`). Cmd+K indexes settings sections and deep-links with a breathe highlight.

## Forbidden

daisyUI, Next.js, Monaco as the home view, marketing empty states, fake “sandboxed” chrome.
