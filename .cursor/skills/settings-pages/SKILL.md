---
name: settings-pages
description: Use when building settings — workspace, provider, policy, plugins, danger zone. Covers sectioned layouts, autosave vs explicit save, and safe destructive actions.
---

# Settings Pages

Work Buddy lesson: **Settings is not a peer of daily tabs.** Control is a configuration surface reached from a gear, not a fifth lab of equal weight.

Structure: section nav (Provider, Workspace, Policy, Plugins, Updates) + cards + explicit Save. Autosave only for single toggles.

Danger zone: checkout loop/policy, unload fiber, apply staged update — AlertDialog, copy that names the risk. Staging is never trusted.

Do not mix "Save" with silent `onBlur` API key writes without a saved/failed toast.
