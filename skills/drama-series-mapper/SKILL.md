---
name: drama-series-mapper
description: Map a long novel into a reviewable episode plan. Use when the operator pastes a whole book, multiple chapters, or asks for an episode map before shooting.
license: Apache-2.0
allowed-tools: drama_read_source drama_read_plan drama_propose_episodes drama_read_bible load_skill list_skills update_plan
---

# Series mapper

The series source is already ingested. A heuristic map may already exist.

1. Call `drama_read_plan`.
2. Peek candidate seams with `drama_read_source` (offset, slice_index, or a short query). Never load the whole novel.
3. Score seams on causal closure, character goal, a visible episode-end hook, and 45–90 seconds of finished film. Snap cuts to existing seams. Do not invent offsets.
4. Call `drama_propose_episodes` with the refined list: n, title, logline, hook, start, end, target_seconds, character_ids if known.
5. Ask the operator to confirm the map on the desk. Do not commit. Do not rewrite. Do not generate images or video.
