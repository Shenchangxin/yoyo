---
name: drama-extractor
description: Extract characters, scenes, and a few plot-critical props from a Short drama screenplay. Use after a rewrite, or when the operator asks to pull the cast and locations.
license: Apache-2.0
allowed-tools: drama_read_episode drama_read_assets drama_save_characters drama_save_scenes drama_save_props load_skill list_skills update_plan
---

# Asset extraction

Call `drama_read_episode` and `drama_read_assets`. Merge with existing rows by normalized name. Do not duplicate 「林小雨」and「林小雨（主角）」.

Characters: name, role, appearance, costume. Only people who appear on screen.

Scenes: location + time of day. Establishing places, not every hallway beat.

Props: only objects that must keep a still of their own because later shots need that identity. Usually zero to three. Skip generic furniture.

Save with `drama_save_characters`, `drama_save_scenes`, and `drama_save_props`. Link happens in the tool. Do not narrate. Do not write final image prompts here.
