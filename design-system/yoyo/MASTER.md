# Yoyo design system (MASTER)

Product: local personal OS client of one Go loop. Harness / RSI is the control tower. The GUI is a workstation, not a chat product and not an IDE.

This file is the skill-layer source for visual direction. Token values and grammar still live in `docs/DESIGN.md` and `frontend/src/styles.css`. If they disagree, **DESIGN.md wins**.

## Verified direction

| Dial | Value | Why |
|---|---|---|
| Style | Swiss Modernism 2.0 + Flat Design | Grid, Inter, one accent, mathematical spacing. No decorations. |
| Typography | Inter Variable (UI) + system mono (evidence) | “Modern Dark Cinema” pairing for developer tools — already in the client. |
| Color | Zinc chrome + sage accent | Color is evidence. Palettes change the paper, not Harbor. Light default is Neutral (true gray); Paper yellow is opt-in. |
| Motion | Subtle (120–180ms, `--ease-out`) | Motion is state. No overshoot on data. |
| Density | 7 / 4pt grid | Operator density. Empty and lab surfaces get one extra step of air. |
| Variance | 4 | Crafted restraint. Asymmetry in the RSI process rail, not in chrome. |

## First principles

1. **The RSI loop is the product.** Overview is the map. Propose → Prove → Promote is a numbered sequence, not four equal admin tabs.
2. **Show the work, hide the chrome.** Hairline lists. Cards only for objects the operator acts on.
3. **Beauty is instrument craft.** CAS four-cell mark, paper grain, optical type, pointer maps. Not Sparkles, gold leaf, or personality marketing.
4. **Say less, in order.** Headline → one-line context → action.

## Do / Don't

- Do: Inter, 4pt grid, zinc + sage, inset highlights, underline tabs, MarkWell on empty still-lifes.
- Don't: gradients as decoration, glows, avatars, spinners on text, cards in cards, headlines > 21px, daisyUI, a second agent loop.

## Stack

React + Tailwind v4 + shadcn/Radix. Semantic CSS variables only. Lucide icons (existing family — do not mix Phosphor into the same layer).
