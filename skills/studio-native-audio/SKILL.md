---
name: studio-native-audio
description: Attach existing CAS audio (voice or BGM) to a shorts project instead of generating new speech. Use when the operator already has a take, a bed, or asks to reuse drama/canvas audio.
license: Apache-2.0
allowed-tools: studio_get studio_update studio_skip_audio studio_apply_timing load_skill list_skills update_plan
---

# Native audio harvest

`studio_get`. Take `narration_hash`, `subtitle_hash`, and `bgm_hash` only from hashes the operator pasted or that already exist on another video object they named. Never mint audio.

`studio_update` those fields. `studio_apply_timing` so shot length follows the take. If they want no voice track, `studio_skip_audio` after the script is approved. Leave assemble to `studio-assemble`. If they need new TTS, say so and `load_skill` `studio-assemble` instead of faking a hash.
