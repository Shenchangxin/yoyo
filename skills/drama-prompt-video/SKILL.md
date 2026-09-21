---
name: drama-prompt-video
description: Write per-shot video prompts for Short drama using @Name references that later resolve to reference-image slots. Use after the board exists and stills are ready.
license: Apache-2.0
allowed-tools: drama_read_storyboard_context drama_update_storyboard load_skill list_skills update_plan
---

# Shot video prompt

Call `drama_read_storyboard_context`. For shots missing `video_prompt`, write a short motion prompt (about three lines): camera, action, lighting, spoken line if any.

Reference on-screen assets with `@Name` using the exact character, scene, or prop names from context. Do not invent `@图片N` yourself; the host rewrites names into slot order.

Patch with `drama_update_storyboard` and only `storyboard_id` plus `video_prompt`. Do not generate video.
