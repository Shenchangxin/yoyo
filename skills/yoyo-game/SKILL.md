---
name: yoyo-game
description: Generate a spatial 3D game world (游戏世界) from a play intent. Use when the operator wants a playable game world, third-person collect/reach, or a Code2Games-style scene plus rules. Not for adapting a novel into HTML.
license: Apache-2.0
allowed-tools: game_create game_list game_get game_update game_analyze_scene game_plan_rules game_extract_candidates game_place game_realize_assets game_compile game_play game_verify game_repair video_route load_skill list_skills read_skill_file update_plan ask_user read_file view_image browser_open browser_act browser_verify browser_screenshot tool_search
---

# Yoyo game world

Orchestrate the **bound** game-world project. The host owns `playable/`. Do not `write_file` `game-worlds/*/index.html` or `playable/runtime.js`. Do not start a static server. Do not invent a 2D canvas runner. Do not `load_skill` novel-to-game.

Stage siblings — `load_skill` the sibling before that stage:

`yoyo-game` · `game-world-analyze` · `game-world-rules` · `game-world-place` · `game-world-build` · `game-world-qa`

1. If a world is already bound, `game_get` then drive that id. Do **not** `game_create` a second world. Only `game_create` when nothing is bound. `video_route` dest=game with title/intent also compiles. If tools are missing, `tool_search` one name at a time.
2. `game_analyze_scene` then `game_plan_rules`.
3. `game_extract_candidates` then `game_place` (host restores xyz).
4. `game_realize_assets` then `game_compile`.
5. `game_play` once, then `game_verify` on the same HTML. Host writes `qa/verification.json`.
6. On FAIL, `game_repair` and verify again.

Workspace: `game-worlds/<slug>/`. Read contracts with `read_skill_file` path=`references/game-json.md`.
