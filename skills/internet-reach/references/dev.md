# GitHub and code

Repo root (`https://github.com/owner/repo`): `web_fetch` returns description, language, stars from the public API.

Issues, blobs, and trees: `web_fetch` the page, or `web_search` with `site:github.com owner/repo query`.

`gh` via `shell` only if `reach_status` lists it. Prefer host fetch for public metadata.

Do not scrape HTML with curl.
