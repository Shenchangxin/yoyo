---
name: game-world-analyze
description: Build a procedural scene packet for a Yoyo game world. Use after the operator wants a spatial playable world. Visual notes only — no game design, no coordinates.
license: Apache-2.0
---

# Analyze scene

Call `game_analyze_scene`. The host writes `ir/scene.json` from a heightfield. Do not invent xyz. Do not tag existing rocks or water as gameplay props.
