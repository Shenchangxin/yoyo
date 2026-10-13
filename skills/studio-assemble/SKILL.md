---
name: studio-assemble
description: Voice the bound shorts script and FFmpeg-assemble shots into an mp4 in the asset library. Use when the operator asks to 成片, 合成, speak, or export the short.
license: Apache-2.0
allowed-tools: studio_get studio_update studio_speak studio_skip_audio studio_apply_timing studio_assemble studio_reframe studio_burnin studio_slideshow studio_wrap studio_package studio_cover studio_copy studio_preflight load_skill list_skills update_plan
---

# Assemble

`studio_get`. Script must be approved. Refuse if shots have no `image_hash` / `video_hash`. Confirm aspect with the operator before changing it. `studio_preflight` the gate you are about to spend.

If narration is missing and a speech adapter exists, `studio_speak` with the script (or a tighter VO), then `studio_apply_timing` so shot length follows the voice. If they already have timings, `studio_skip_audio`. Then `studio_assemble`. Report the CAS hash and that the file landed in 短片成片.

Optional ops, only if asked:

- `studio_reframe` for 9:16 / 16:9 / 1:1
- `studio_burnin` with an SRT CAS hash
- `studio_slideshow` from image hashes
- `studio_wrap` to burn a title onto the cut (FFmpeg talking-head wrap; Remotion is never the default)
- `studio_cover` / `studio_copy` / `studio_package` to build the local delivery pack. Packaging does not post.

If FFmpeg or speech is missing, point to Settings → Generation. Do not shell out to ffmpeg. Never invent hashes.
