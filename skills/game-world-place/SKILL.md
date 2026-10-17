---
name: game-world-place
description: Extract host-owned placement candidates and pick candidate ids for a Yoyo game world. Models never output world coordinates.
license: Apache-2.0
---

# Place

`game_extract_candidates` then `game_place`. The model may only choose existing `candidate_id` values. The host restores xyz and spacing.
