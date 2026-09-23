---
name: drama-prompt-video
description: Write per-shot video prompts for Short drama using @Name references that later resolve to reference-image slots. Use after the board exists and stills are ready.
license: Apache-2.0
allowed-tools: drama_read_storyboard_context drama_update_storyboard load_skill list_skills update_plan
---

# Shot video prompt

Call `drama_read_storyboard_context`. For shots missing `video_prompt`, write the motion prompt and patch with `drama_update_storyboard` using only `storyboard_id` plus `video_prompt`. Do not generate video. Do not rewrite the board.

## Shape

1. First line is an info header: the on-screen people and the scene, each referenced as `@Name` using the exact names from context.
2. Then one line per about three seconds. Camera, action, light, spoken line if any.
3. Map each `【镜头N】` in the shot `description` to one or two consecutive 3-second lines. Keep order. Do not invent subshots. Do not invent dialogue — pull lines from the matching `【镜头N】` (`角色名说：「…」` or `旁白：…`).
4. Cuts inside a shot may change size/angle/subject. They must not change scene. Align cut points with `【镜头N】`.

Reference on-screen assets with `@Name` using the exact character, scene, or prop names from context. Do not invent `@图片N` yourself; the host rewrites names into slot order.
