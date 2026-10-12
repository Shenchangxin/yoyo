---
name: studio-assemble
description: Voice the bound shorts script and FFmpeg-assemble shots into an mp4 in the asset library. Use when the operator asks to 成片, 合成, speak, or export the short.
license: Apache-2.0
allowed-tools: studio_get studio_update studio_speak studio_assemble studio_reframe studio_burnin studio_slideshow studio_wrap load_skill list_skills update_plan
---

# Assemble

`studio_get`. Refuse if shots have no `image_hash` / `video_hash`. Confirm aspect with the operator before changing it.

If narration is missing and a speech adapter exists, `studio_speak` with the script (or a tighter VO). Then `studio_assemble`. Report the CAS hash and that the file landed in 短片成片.

Optional ops, only if asked:

- `studio_reframe` for 9:16 / 16:9 / 1:1
- `studio_burnin` with an SRT CAS hash
- `studio_slideshow` from image hashes
- `studio_wrap` to burn a title onto the cut (FFmpeg talking-head wrap; Remotion is never the default)

If FFmpeg or speech is missing, point to Settings → Generation. Do not shell out to ffmpeg.
