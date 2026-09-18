# Yoyo design

Operate-mode workstation. Steal Codex density, Linear surface craft, Work Buddy settings authority. Do not clone any of them.

## World

- **Dark-first** cool zinc (`#0f1011` canvas, `#161718` sidebar). Light mode uses the same semantic tokens on warm paper.
- **Muted sage** (`#6d8a7e` dark / `#5d7368` light) is a semantic token, not chrome. Buttons, tabs, inputs, and focus rings stay zinc / foreground. No accent underlines, no colored input halos.
- Inter Variable with `cv11` + `ss03`, optical sizing. System monospace for diffs, hashes, and tool payloads.
- Radius scale: chrome `--radius` (10px), composer `--radius-composer` (16px).
- Material ladder via hairline overlays (`--border` is a white/black mix) and inset top highlights (`--shadow-card`, `--shadow-composer`). No decorative glow.

## Tokens only

Features use `bg-background`, `bg-panel`, `bg-sidebar`, `bg-card`, `bg-lift`, `text-foreground`, `text-muted`, `border-border`, `text-danger`. Reserve `text-accent` for evidence (diff add, Harbor pass), never for chrome. No raw hex in feature files. Theme lives in `frontend/src/styles.css` (`html.dark` / `html.light`) plus `color-scheme`.

## Layout

Padded app frame (`p-2`). Sidebar is a rounded card with inset highlight: CAS mark + wordmark + active hash chip, New chat, search, recency-grouped threads, Skills then Harness rows, notifications + settings gear. Skills is a first-class rail module (installed packs + Work Buddy catalog); it keeps the thread list and occupies the main column. Main column has a quiet drag `PageHeader`. Three resizable panes on Agent. Review collapses to a sheet under 1100px. Sidebar becomes an overlay under 800px. Harness is one RSI workspace (Overview / Propose / Prove / Promote); chat docks as an optional right rail, off by default. Settings occupies the main column and hides the thread rail (200px nav + 680px content).

## Transcript and composer

- Transcript uses the main pane (gutters only). Composer stays a centered 48rem column and is the brightest object (inset sheen, zinc send). User bubbles right; assistant prose left, max ~75ch.
- Process (tools, reasoning, injections) folds per model round: live shows the in-flight step, settled collapses to “N tools”. A turn is one letter — one copy for all assistant prose, process lines are not selectable. Outcomes stay open: assistant prose, patch/artifact cards, approvals. Streaming caret only while tokens arrive. Stick-to-bottom yields when the user scrolls up.
- Enter sends. Shift+Enter newline. Running turns expose Steer next to Stop. Composer disabled with an inline reason until a workspace exists.
- Empty state: operator sentence + three starter rows. No Sparkles headline, no ChatGPT pills.

## Skills

Left-rail module, not a Review tab. Installed packs are a source-grouped list (bundled / home / workspace / market). Market fetches the Work Buddy public catalog (`infometa/workbuddyskills`) at runtime and installs `SKILL.md` only into `~/.yoyo/skills` after a static scan. Catalog UI is zinc hairline sections and rows — one outer radius per category, `divide-y` inside, outline install buttons. No app-store tiles, no accent chrome.

## Handoff

Thread menu: pop-out window (`/?popout=`), open workspace in editor, open system terminal. Review files and artifact cards can open a path in the OS. HTML tool results render in a sandboxed iframe; PDFs open in the OS. Isolation chrome shows the real kind (`job object`, Seatbelt) and never a sandbox badge unless `Sandbox` is true.

## Review

Split or unified highlighter. File list from the git diff. Apply selected with an undo toast. Context meter uses tabular nums and layer chips (trusted pin vs untrusted tools). Last inspector tab is remembered per thread.

## RSI labs

Harbor is a three-beat story (held-in, held-out, safety veto) plus a verdict. Evolve archive is a parent-linked version timeline with this-cycle playbook deltas. Promote is a checkout ceremony: pointers as a list, staging vs active field diff first, arbitrary hashes behind a disclosure.

## Motion and a11y

`prefers-reduced-motion` disables caret blink, section breathe, and decorative transitions. Visible focus rings on controls. Keyboard: New chat, send, stop, palette, Once/Deny, toggle review, settings (`Mod+,`). On Harness, `Mod+1`–`Mod+4` switch Overview / Propose / Prove / Promote. Cmd+K indexes settings sections and deep-links with a breathe highlight.

## Forbidden

daisyUI, Next.js, Monaco as the home view, marketing empty states, fake “sandboxed” chrome, zero-offset accent halos, tracked-out ALL-CAPS eyebrows.
