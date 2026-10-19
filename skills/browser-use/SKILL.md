---
name: browser-use
description: Drive isolated Chrome accurately. Use when navigating, clicking, filling forms, waiting on SPAs, managing tabs, or recording a web workflow. Snapshot before every click; prefer ref= handles.
license: Apache-2.0
allowed-tools: browser_open browser_snapshot browser_click browser_type browser_fill browser_scroll browser_hover browser_select browser_press browser_act browser_verify browser_screenshot browser_takeover browser_tabs browser_profile browser_record browser_cookies load_skill list_skills
---

# Isolated browser (browser-use)

You drive a real Chromium over CDP. The Wails window is the operator UI, not the page. Clicks that miss usually mean a stale handle or a covered node — not a need to spawn Chrome or `python -m http.server`.

## Contract

1. **Snapshot before every click.** Refs (`btn_0`, `inp_1`) expire on navigation. After `browser_open`, a tab switch, or a click that changes the URL, call `browser_snapshot` again.
2. **Prefer `ref=`.** The snapshot prints `[ref=btn_0]  #opt-0  <button> "Start"`. Pass `ref=btn_0` (or `selector="ref=btn_0"`). CSS and `text=` are fallbacks.
3. **Never hardcode coordinates.**
4. **Do not reload a live page** to retry a click. `browser_click` / `browser_act` / `browser_screenshot` do not navigate. A second `browser_open` of the same file wipes in-page state.
5. **Forms: `browser_fill`.** It replaces. `browser_type` appends real key events and is for games.
6. **Login / MFA / CAPTCHA: `browser_takeover`.** That opens a headed window and records the operator's flow into a skill. Do not paste secrets. Named profiles (`browser_profile` `use` `github`) persist cookies afterward.
7. **On miss:** read the listed live actions. The host already retries by testid / accessible name. Pick a `ref=` from the new list; do not invent `nth-of-type` paths.

## Observation levels

`browser_snapshot` `level`:

| level | when |
|---|---|
| `minimal` | cheap URL/title check after a CDP error |
| `interactive` | default for remote http(s) — refs only |
| `full` | default for workspace HTML / games — visible text + refs |
| `structured` | JSON for scripts |

## Multi-step paths

Use `browser_act` instead of one click per turn when the path is known:

```
click / type / fill / wait / assert / screenshot / open / snapshot / scroll / hover / select / press / tab
```

`wait` with `selector`, `contains`, or `url` **polls**. A bare `wait` sleeps (max 5s). Condition waits may run up to 15s.

Off-screen targets: `scroll` first, or pass the `ref=` so the host `scrollIntoView`s it. A "no hit-test element at its centre" error means the node is clipped — scroll, then snapshot.

## Profiles and tabs

- Isolated profile `default` is the workspace/game profile.
- `browser_open` `profile=github` or `browser_profile` `use` `github` keeps that site's cookies. Never attach the operator's daily Chrome unless the operator asked for `lane=attached`.
- OAuth popups: `browser_tabs` `list`, then `switch`.

## Recording

`browser_takeover` starts a recording. After the operator confirms, Yoyo writes a draft skill under `.yoyo/skills/`. You can also `browser_record` `start` / `stop`. Secrets become `{{sensitive_value}}` — ask the operator; do not log them.

## Forbidden

- `file://` or `data:` URLs
- Starting `python -m http.server` or a second Chrome
- Connecting to CDP 9222/9333 unless `lane=attached` was requested
- Clicking Publish on a creator site (`studio-publish` handles handoff)
- Pasting `sk-`, `api_key`, or PEM blocks into the page
