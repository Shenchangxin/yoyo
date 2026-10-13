---
name: studio-storyboard
description: Turn a shorts script into storyboard shots that point at CAS image_hash or video_hash. Use after the script exists, or when the operator asks for 分镜 / 镜头.
license: Apache-2.0
allowed-tools: studio_get studio_update studio_generate_shot studio_generate_missing studio_set_identity studio_catalog studio_preflight load_skill list_skills update_plan
---

# Studio board

`studio_get`. The script must be `approved` before spending image. `studio_preflight` `visuals` if unsure the image adapter is configured.

Split the spoken script into 3–8 shots. Each shot:

- `caption`: on-screen or spoken line for that beat
- `visual` / `prompt`: picture suggestion, not a camera essay
- `duration`: 2–5 seconds (or keep timings from Speak)
- `motion`: `ken-burns` for stills unless the operator asks otherwise
- `image_hash` or `video_hash`: only real CAS hashes

`studio_update` with `shots`. To fill missing stills, `studio_generate_shot` / `studio_generate_missing` — never invent hashes. If generation is not configured, list the stills the operator must drop onto the desk. Do not call `studio_assemble` here.
