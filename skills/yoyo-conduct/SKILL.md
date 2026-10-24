---
name: yoyo-conduct
description: Failure playbook for a repeated tool error, stale network report, elided result, chunked write, or browser reload.
---

# Yoyo conduct

Load this when a tool keeps failing, a result was elided, or a write or browser step is going in circles. Do not load it for a fresh task that is proceeding.

## Repeated errors

If the same tool error repeats (blocked network, capability, path jail, HTTP 406, invalid JSON arguments), stop retrying that call and finish with what you already have.

## Network

Public HTTP uses `web_fetch` / `web_search`, not shell curl or python urllib. A leftover workspace report that says the network is dead is stale. Try `web_fetch`. If https returns 406, the tool retries http.

## Files

A file already on disk from an earlier session (`list_dir` mtime before this turn) is leftover, not this turn's write. After `write_file` fails, do not treat that path as success, and do not `python -c` the report body.

`write_file` accepts at most 32 KB per call. Write a long file as create/overwrite, then `mode=append` parts. A cut-off call is refused and nothing is written.

Cite paper ids only as they appear in tool output.

## Skills

Skill helpers live in the pack: `read_skill_file` and `run_skill_script`. Do not glob workspace `scripts/` for them, and do not wait-loop a hung process after the idle/block fuse.

## Browser

`browser_open` reloads the page and wipes clicks. Do not call it again for the same file unless the file changed. `browser_act`, `browser_screenshot`, and `browser_click` use the current page and do not reload. Snapshot first. Prefer `ref=` handles from the snapshot.

## Checks

When a check FAILs, edit the cited product file so that same check can pass. A `node -e` probe or a rewrite of `qa/verify` is not the deliverable. Progress is an observation that changed.
