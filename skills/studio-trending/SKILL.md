---
name: studio-trending
description: Fetch live Chinese social hot lists and turn promising rows into shorts topic cards. Use when the operator asks for 行业热点, 热搜, Weibo/Douyin/Zhihu trends, or what to make a short about today.
license: Apache-2.0
allowed-tools: trends_fetch topic_save topic_list topic_get load_skill list_skills update_plan
---

# Studio trends

Call `trends_fetch`. Default platforms: weibo, douyin, zhihu. Add bilibili, toutiao, baidu only if asked. Do not scrape platform homepages. Do not invent rankings.

Read titles, heat, and URLs. Pick a few that can become a 15–60s short for this operator. For each pick, `topic_save` with:

- `title`: the hot-list wording, trimmed
- `source_platform` / `source_title`
- `format`: `short`
- `status`: `inbox`
- `note`: heat plus why it is filmable in one sentence

Do not score yet. `load_skill` `studio-topic-eval` after the operator chooses a card. Do not narrate the JSON. Summarize picks in the operator's language.
