---
name: studio-storyboard
description: Turn a shorts script into storyboard shots that point at CAS image_hash or video_hash. Use after the script exists, or when the operator asks for 分镜 / 镜头.
license: Apache-2.0
allowed-tools: studio_get studio_update load_skill list_skills update_plan
---

# Studio board

`studio_get`. Split the script into 3–8 shots. Each shot:

- `caption`: on-screen or spoken line for that beat
- `duration`: 2–5 seconds
- `prompt`: still/clip prompt if no hash yet
- `image_hash` or `video_hash`: only real CAS hashes the operator supplied or that already sit on the project. Never invent hashes.

Ken Burns stills use `motion: ken-burns`. `studio_update` with `shots`. If hashes are missing, list the stills the operator must drop onto the desk. Do not call `studio_assemble` here.
