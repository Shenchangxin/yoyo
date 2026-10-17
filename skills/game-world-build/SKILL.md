---
name: game-world-build
description: Realize primitives or optional textures and compile a sealed Yoyo playable. Do not rewrite runtime.js.
license: Apache-2.0
---

# Build

`game_realize_assets` then `game_compile`. Logical types stay markers. The compiled HTML is the playable. Do not `write_file` `playable/runtime.js`, `playable/index.html`, or `game-worlds/*/index.html`.
