---
name: yoyo-desktop-client
description: Product IA and UX rules for the Yoyo Wails desktop client. Use when changing frontend shell, transcript, composer, inspector, labs, Control, or desktop chrome. Enforces harness-workstation grammar (Codex/Work Buddy/Claude), not a Cursor IDE clone and not a ChatGPT marketing skin.
---

# Yoyo desktop client

GUI is a **client of one Go harness**. Do not add a second agent loop, LangChain, Vercel AI SDK, or assistant-ui runtime. Bindings and `lib/stream.ts` stay the live protocol.

## IA (non-negotiable)

```
[ Thread rail ] [ Transcript + composer ] [ Review ]
                      Labs are secondary
```

- Thread is an object: create, rename, fork, search, running indicator.
- Plan is a composer chip, not a page. The chip starts collapsed; the
  in-progress step stays on the header.
- Approvals render **in the stream** (Claude grammar). Inspector is review, not the only place to click Once/Deny.
- Harbor / Evolve / Harness are laboratory workspaces, not chat siblings of equal weight.
- Skills is a left-rail module (installed + market), not a Review tab and not a settings page.
- Control/settings is a gear surface (Work Buddy), not a fifth primary lab.

Steal, do not clone:

| Source | Steal | Leave |
|---|---|---|
| Codex | Thread-centric shell, floating composer, review in-thread, skills later | Cloud multi-agent home, personality marketing |
| Work Buddy | Settings as control plane, surgical event updates, chat as right rail when in labs | Flask dashboard tabs, Telegram |
| Claude Code | Plan chip, in-stream permission cards | VS Code editor |
| Cursor | Three-pane density, thread object | IDE, LSP, composer-as-editor |

## Visual world

The full system lives in `docs/DESIGN.md` — read it before touching any surface. The one-line version: **show the work, hide the chrome; color is evidence; motion is state.**

- One conversation column shared by transcript and composer (`lib/thread.ts`). It tracks the stage (`min(100%, --thread-measure)`); scroll container stays pane-wide.
- Hairline lists, not bordered cards. Cards only for objects the operator acts on (approval, artifact, error).
- Semantic tokens only (`styles.css`): zinc chrome; `success`/`danger` for diff and step state, `warning` for interrupted/needs-approval, `accent` for primary selection. Step opacity for greys.
- Live grammar: shimmer verb + pulse dot + `Worked for` clock while running; past tense and a static glyph the instant it stops. A call without a result on a finished turn is *interrupted*, never spinning.
- Composer: Enter sends, Shift+Enter newline. Context meter appears only once tokens exist.
- Review panel: text tabs with underline, 36px icon toolbars, file-grouped diff with a footer CTA only when something is selected. Selected file preview fills the remaining pane (not a 28rem island); source is Shiki-highlighted.
- Code and diffs in `--font-mono` with ligatures off; counts in `tabular-nums`.

## Engineering

- Split `App.tsx` god state.
- Resizable panes, persisted.
- Virtualize transcript and thread list.
- cmdk palette; Radix dialogs; sonner; RHF+zod on Control.
- Overlay titlebar + drag regions on Windows/macOS.
- Control/settings is a gear surface with tab/section registry, not a fifth lab.
- Wails Events remain source of truth; 1.5s poll only while running.

## Forbidden

- Cursor clone (file tree as home, IDE chrome).
- L4 UI. Evaluator/vault/updater as agent-editable.
- Fake sandbox badges.
- daisyUI (conflicts with shadcn tokens already in use).
- Avatars in the transcript, spinners on text, cards inside cards, labelled buttons in toolbars, ligatures in code (see `docs/DESIGN.md` §9).
