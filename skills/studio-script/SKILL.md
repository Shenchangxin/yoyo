---
name: studio-script
description: Write a short-form shorts script (hook, body, closer) onto the bound project. Use after a topic is ready, or when the operator asks for 口播稿 / 短视频脚本.
license: Apache-2.0
allowed-tools: studio_get studio_create studio_update topic_get load_skill list_skills update_plan
---

# Short script

If no project is bound, `studio_create` from `topic_id` after confirming aspect (default 9:16). `studio_get`. Pull title/angle from `topic_get` when `topic_id` is set.

Write 80–180 Chinese characters (or the operator's language) as:

1. Hook (0–3s), one concrete image
2. Body, two beats max
3. Closer, one line to save or try

No scene headings. No camera essay. Spoken text only. `studio_update` with `script`. Do not assemble. Do not invent CAS hashes.
