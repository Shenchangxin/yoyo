# Video

YouTube and Bilibili watch URLs: `web_fetch` first. The host returns title/metadata and YouTube captions when the timedtext API allows it.

If captions are missing:

- Isolated browser can open the page; do not download the media unless the operator asked.
- `reach_status` may list `yt-dlp` or `bili`. Only then may you `shell` those binaries. Write outputs inside the workspace, never `/tmp`.
- Bilibili: do not use yt-dlp (upstream retired it after 412). Prefer `bili` if listed, else browser.

Do not fetch the whole video to "read" it.
