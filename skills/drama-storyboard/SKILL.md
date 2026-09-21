---
name: drama-storyboard
description: Split a Short drama screenplay into 8–15 second video jobs with 2–4 subshots each. Use when the operator starts the board stage.
license: Apache-2.0
allowed-tools: drama_read_storyboard_context drama_save_storyboards drama_update_storyboard load_skill list_skills update_plan
---

# Segment storyboard

Call `drama_read_storyboard_context`. Honor the duration plan: total seconds ≈ character count / 500 * 60; segment count ≈ total / 12.

Each saved row is one video job:

- 8–15 seconds. Transitions 8–10, narrative 10–15, peak beats 12–15.
- Two to four subshots inside `description`.
- Dialogue floor: spoken characters / 4.5 + 2 seconds, then clamp.
- Bind `scene_id` and `character_ids` / `prop_ids` by name or id from the context.
- `video_prompt` may stay empty; a later pass fills @name references.

First batch: `drama_save_storyboards` with `replace_existing` true, at most 8 rows. Continue with `replace_existing` false until the episode is covered. Do not narrate.
