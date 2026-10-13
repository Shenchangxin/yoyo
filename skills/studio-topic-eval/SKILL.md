---
name: studio-topic-eval
description: Score a shorts topic on seven dimensions and write an angle plus format. Use for 洞察选题, 选题评分, or when choosing which inbox card to produce.
license: Apache-2.0
allowed-tools: topic_get topic_save topic_list studio_expand_topic studio_choose_option studio_persona_template load_skill list_skills update_plan
---

# Topic evaluation

`topic_get` the card. If `note` is thin, `load_skill` `studio-insight` first. If `persona_page_id` is empty and they want a voice check, `load_skill` `studio-persona` — persona is a Pages template, not a session Profile.

Score 1–5 integers in `scores`:

| key | meaning |
|---|---|
| heat | How hot the source list is right now |
| timing | Will it still matter in 48 hours |
| uniqueness | Can this account say it differently |
| feasibility | Can a 9:16 short be assembled from stills/clips/voice we already have |
| persona | Fit to the operator's on-screen voice |
| conversion | Save / follow / comment pull |
| risk | Invert: 5 = safe, 1 = legal/brand hazard |

Write `angle` as one shooting sentence, `format` as `short` / `talking-head` / `list`, and set `status` to `ready` only when feasibility ≥ 3 and risk ≥ 3. Otherwise leave `inbox` and say why. Optional `studio_expand_topic` then `studio_choose_option` if they want 3–5 angles before locking — that does not replace scores. `topic_save`. Do not create a Studio project until the operator asks.
