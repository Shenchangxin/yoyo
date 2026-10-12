---
name: studio-insight
description: Research a shorts topic with web search and write a knowledge insight onto the topic card. Use for 知识洞察, 背景检索, 为什么这个热点成立, or when a topic needs facts before scoring.
license: Apache-2.0
allowed-tools: web_search web_fetch topic_get topic_save topic_list load_skill list_skills update_plan
---

# Knowledge insight

`topic_get` (or `topic_list` if the id is missing). Search with `web_search`, then `web_fetch` one or two primary sources. Prefer public reporting over blogs.

Write `note` as:

1. What happened (one sentence, dated if possible)
2. Why it is spreading
3. One concrete angle a short can take
4. Source URLs

Keep scores untouched. Do not turn the topic into a personal-OS idea. If search is unconfigured, say so and stop — keys live in Settings → Generation → Search. Patch with `topic_save`.
