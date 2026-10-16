---
name: cut-edit
description: Analyze a source clip and cut highlights with audio kept. Use when the operator asks to 拆解, 精剪, 切三条, or ingest a talking-head / lecture.
license: Apache-2.0
allowed-tools: cut_get cut_ingest cut_analyze cut_propose cut_apply cut_extract load_skill update_plan ask_user
---

# Cut

`cut_get` or `cut_ingest` with a real CAS hash. `cut_analyze` (scene + silence). `cut_propose` count=3 window=15-30 unless they asked for one tight cut.

Report clip scores (energy / completeness). Confirm before `cut_apply`. The result keeps the soundtrack.

Do not equal-slice. Do not strip audio. Do not call studio_generate_shot. If they then want 配音, `load_skill` cut-dub.
