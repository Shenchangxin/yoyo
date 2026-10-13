---
name: studio-script
description: Write a short-form shorts script (hook, body, closer) onto the bound project. Use after a topic is ready, or when the operator asks for 口播稿 / 短视频脚本.
license: Apache-2.0
allowed-tools: studio_get studio_create studio_update studio_approve_script studio_catalog studio_apply_template studio_apply_preset studio_set_identity studio_expand_topic studio_choose_option topic_get load_skill list_skills update_plan
---

# Short script

If no project is bound, `studio_create` from `topic_id` after confirming aspect (default 9:16). Optional `template_id` / `preset_id` from `studio_catalog` locks frame, tone, and character contract. `studio_get`. Pull title/angle from `topic_get` when `topic_id` is set. If the card has no locked angle, offer `studio_expand_topic` then `studio_choose_option`.

Write 80–180 Chinese characters (or the operator's language) as spoken text only:

1. Hook (0–3s), one concrete image
2. Body, two beats max
3. Closer, one line to save or try

No scene headings. No camera essay. `studio_update` with `script`, plus `hook` / `cta` / `platform` / `tone` when known. Then `studio_approve_script` only after the operator confirms the spoken line. Do not assemble. Do not invent CAS hashes. Do not call Speak or image gen until the script is approved.
