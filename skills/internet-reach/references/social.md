# Social (login)

Twitter/X, 小红书, Reddit, Facebook, Instagram have no zero-config public search on Yoyo.

Path:

1. `reach_status`. If a platform CLI is listed (`twitter`, `bili` is video, not this file), you may `shell` that CLI with the operator's existing env. Do not print cookies.
2. Otherwise isolated Chrome: `browser_profile` `use` a named profile, `browser_takeover` for login. Never attach the daily Chrome unless the operator asked for `lane=attached`.
3. Stop if the site shows an environment-risk / captcha loop. Ask the operator.

Do not export Cookie-Editor blobs into the workspace. Do not put tokens in the transcript.

Ban risk is real on scripted social APIs. Prefer a dedicated account the operator names.
