# Yoyo design

Operate-mode workstation. Steal Codex density and Work Buddy settings authority. Do not clone either product.

## World

- **Dark-first** zinc surface. Light mode exists via the same semantic tokens.
- **One teal accent** for primary send and selection (`--accent`).
- Inter Variable for UI. System monospace for diffs, hashes, and tool payloads.
- Radius scale: controls `--radius`, composer `--radius-composer`. No extra elevation fashion.

## Tokens only

Features use `bg-background`, `bg-panel`, `bg-sidebar`, `bg-lift`, `text-foreground`, `text-muted`, `border-border`, `text-accent`, `text-danger`. No raw hex in feature files. Theme lives in `frontend/src/styles.css` (`html.dark` / `html.light`) plus `color-scheme`.

## Layout

Three resizable panes. Review collapses to a sheet under 1100px. Overlay titlebar with drag regions; form controls are `no-drag`. Labs keep the thread rail; chat docks as a right rail.

## Transcript and composer

- User bubbles right; assistant prose left, max ~75ch.
- Tools as compact disclosure rows. Streaming caret. Stick-to-bottom yields when the user scrolls up.
- Enter sends. Shift+Enter newline. Composer disabled with an inline reason until a workspace exists.
- Empty state: three operator prompts + workspace CTA. No Sparkles headline.

## Review

Split or unified highlighter. File list from the git diff. Apply selected with an undo toast. Context meter uses tabular nums and layer chips (trusted pin vs untrusted tools).

## Motion and a11y

`prefers-reduced-motion` disables caret blink and decorative transitions. Visible focus rings on controls. Keyboard: New chat, send, stop, palette, Once/Deny, toggle review.

## Forbidden

daisyUI, Next.js, Monaco as the home view, marketing empty states, fake “sandboxed” chrome.
