---
name: settings-pages
description: Use when building settings — workspace, provider, policy, plugins, danger zone. Covers sectioned layouts, autosave vs explicit save, and safe destructive actions.
---

# Settings Pages

Work Buddy lesson: **Settings is not a peer of daily tabs.** Control is a configuration surface reached from a gear, not a fifth lab of equal weight.

Structure: tab + section registry (`features/settings/registry.ts`) + split view (200px nav, max 680px content) + `SettingSection` / `SettingRow` cards. Cmd+K indexes sections and deep-links with a breathe highlight.

Autosave only for single toggles. Secrets, provider fields, and budgets use explicit Save + toast.

Danger zone: unload fiber, apply staged update — ConfirmDialog / AlertDialog, copy that names the risk. Staging is never trusted.

Do not mix "Save" with silent `onBlur` API key writes without a saved/failed toast.
