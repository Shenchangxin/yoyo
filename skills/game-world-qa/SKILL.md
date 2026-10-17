---
name: game-world-qa
description: Play and verify a compiled Yoyo game world with isolated Chrome. Host writes qa/verification.json.
license: Apache-2.0
---

# QA

`game_play` opens the workspace HTML once. Then `game_verify` with that same path. Do not hand-write PASS. Do not start `python -m http.server`. On FAIL call `game_repair`.
