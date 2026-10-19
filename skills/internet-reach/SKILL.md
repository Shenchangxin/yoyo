---
name: internet-reach
description: >
  Use when the operator wants internet research, 调研, 全网搜索, reading a URL
  as clean text, YouTube/Bilibili transcripts, GitHub/V2EX public pages, or to
  see which search backends are live. Host tools only. NOT for local grep,
  writing the report itself, posting, or driving the daily Chrome.
license: Apache-2.0
allowed-tools: web_search web_fetch reach_status browser_open browser_snapshot browser_act browser_takeover browser_profile cite_sources load_skill list_skills read_skill_file update_plan ask_user task
---

# Internet reach (host tools)

Yoyo already searches and fetches. This skill is the routing table, not a second runtime. **Do not curl. Do not invent search APIs.**

## Contract

1. `reach_status` once if the task spans more than the public web (video, GitHub, login platforms).
2. Public web: `web_search` then `web_fetch` primary URLs. Cite with `cite_sources`.
3. A query that is already a URL: `web_search` (it fetches) or `web_fetch` directly.
4. Login walls: isolated `browser_takeover` + named `browser_profile`. Never the daily Chrome. Never CDP 9222 unless the operator asked for `lane=attached`.
5. Optional `agent-reach` CLI appears only in `reach_status`. Use those CLIs via `shell` only for the named platform. Still never curl, never mcporter for generic search.

## Route

| Intent | Do |
|---|---|
| Web search / 调研 | `web_search` → `web_fetch` 1–3 sources. Details: `references/search.md` |
| Read a URL / RSS | `web_fetch`. Details: `references/web.md` |
| YouTube / B站 / captions | `web_fetch` the watch URL first. Details: `references/video.md` |
| GitHub repo / code search | `web_fetch` the repo URL; code search is `web_search` with `site:github.com`. Details: `references/dev.md` |
| Twitter / 小红书 / Reddit / FB / IG | Isolated browser + operator login. Details: `references/social.md` |
| Jobs / LinkedIn | `web_search` / `web_fetch` public pages; login via `browser_takeover`. Details: `references/career.md` |

## Forbidden

- `curl`, `wget`, `Invoke-WebRequest`, `mcporter` for generic search/read
- `https://r.jina.ai/…` or `https://s.jina.ai/…` via shell — the host already uses Jina
- OpenCLI / daily Chrome / Boss CDP `127.0.0.1:9222`
- Replacing `web_search` with a new tool name
- MUST-USE claims that skip host tools

Read a reference with `read_skill_file` (`skill=internet-reach`, `path=references/…`) only when this table is not enough.
