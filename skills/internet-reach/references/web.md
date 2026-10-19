# Read pages

`web_fetch` returns extracted text. RSS/Atom is compacted. YouTube, Bilibili, V2EX, and GitHub repo roots are structured.

If the page is a login wall or a captcha:

1. Do not curl Jina yourself. The host already retries thin/blocked HTML via Jina Reader (not for loopback).
2. Isolated `browser_open` / `browser_snapshot`. Login/MFA: `browser_takeover` and a named `browser_profile`.

Loopback (`http://127.0.0.1:…`) is in scope for the operator's local servers. Private LAN and cloud metadata are blocked.

Never wrap URLs as `https://r.jina.ai/…` in shell.
