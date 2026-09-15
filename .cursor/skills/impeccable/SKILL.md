---
name: impeccable
description: Designs and iterates production-grade frontend interfaces. Use when the user wants to design, redesign, shape, critique, audit, polish, clarify, distill, harden, optimize, adapt, animate, colorize, extract, or otherwise improve a frontend interface. Covers product UI, app shells, components, forms, settings, onboarding, empty states, UX review, visual hierarchy, IA, a11y, motion, theming, and design systems. Not for backend-only work.
---

# Impeccable

Operate as an award-winning design director: production-grade code, a clear point of view, and exceptional craft. Yoyo is an **Operate** surface (desktop harness workstation), not a marketing page.

This project vendored the skill text and the two references that matter for product UI. The upstream Impeccable CLI/scripts are not required.

## Setup

1. Read [craft-floor.md](craft-floor.md) immediately before any UI edit.
2. For app chrome, settings, transcripts, composers, and labs, also read [operate.md](operate.md).
3. Honor Yoyo product constraints in `yoyo-desktop-client`. The GUI is a client of one Go harness. Do not clone Cursor IDE. Do not invent a ChatGPT landing-page look when the brief asks for Codex/Work Buddy workstation craft.

## Modes

- **Operate** (default for this repo): scanability, native affordances, density, consistency. Brand lives in precise details.
- **Persuade / Experience / Read**: only if the surface is actually marketing, a gallery, or docs.

## Commands (use as review lenses)

| Lens | When |
|---|---|
| shape | Plan UX before code |
| critique | Heuristic UX scoring |
| audit | a11y, perf, responsive |
| polish | Final pass before ship |
| quieter | Tone down AI-slop chrome |
| distill | Remove complexity |
| harden | Errors, empty, i18n, edges |
| onboard | First-run and empty states |
| typeset / layout | Type scale and spacing |
| clarify | Copy, labels, errors |

## How to design here

- The brief wins. Yoyo already pinned: three-pane agent client, thread as object, Plan as composer chip, labs secondary, Harbor owns promotion.
- Refinement preserves identity. Do not "improve" by swapping the zinc workstation for gold lacquer, cream serif, or acid-green templates.
- Product UI fails as strangeness without purpose: mismatched controls, Sparkles empty states, gradient fades, identical 28px radii on everything.
- Verify in one batched pass (desktop density + narrow window). Fix in one batch. Stop.

## Absolute floors

See [craft-floor.md](craft-floor.md). Highlights for this codebase:

- Visible `:focus-visible` rings; never `outline-none` without a replacement.
- Hover, disabled, loading, error, empty on every interactive region.
- Theme caret, selection, scrollbars, and tabular nums.
- No kicker/eyebrow labels. No card-of-cards. No emoji as UI.
- Motion 150–250ms, state only, honor `prefers-reduced-motion`.
