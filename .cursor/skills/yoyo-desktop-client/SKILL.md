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
- Plan is a composer chip, not a page.
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

## Visual world (Operate)

- Dark zinc workstation. One teal accent for primary send/selection. Distinctive craft comes from **review density and lab evidence**, not Sparkles or gold leaf.
- Semantic tokens only. Theme selection, caret, scrollbars, `color-scheme: dark`.
- Composer: portal-grade focus, Enter-to-send (Shift+Enter newline) **or** document Ctrl+Enter consistently; Codex uses Enter.
- User bubbles right; assistant prose left, max ~75ch; tools as compact disclosure rows.

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
