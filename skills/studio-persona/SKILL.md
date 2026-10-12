---
name: studio-persona
description: Create a creator-voice page (Pages template) and link it on a shorts topic. Use for 人设, 口吻, 账号定位. This is not a session Profile.
license: Apache-2.0
allowed-tools: studio_persona_template studio_persona_create topic_get topic_save load_skill list_skills update_plan
---

# Creator persona (Pages, not Profile)

Profiles are capability projections. The video channel ignores them. Persona lives as a markdown page.

If they already have a page id, `topic_save` with `persona_page_id` and stop.

Otherwise `studio_persona_create` with a short `name` (account or show name) and `topic_id` when a card is in hand. Tell the operator the page is in Pages and they should fill 定位 / 六维 / 样本口播 before scoring `persona` on the topic.

Do not write the persona into Profile instructions. Do not treat `personal.ideas` as a topic.
