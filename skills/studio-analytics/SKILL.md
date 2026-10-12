---
name: studio-analytics
description: Open a platform analytics page and record operator-pasted reads/likes/comments onto a shorts publish log. Use for 数据, 播放量, 复盘. Yoyo does not scrape dashboards.
license: Apache-2.0
allowed-tools: studio_publish_list studio_publish_analytics studio_publish_mark topic_get topic_save load_skill list_skills update_plan
---

# Studio analytics (paste, not scrape)

`studio_publish_list`. Pick the posted (or opened) row. `studio_publish_analytics` opens the vendor dashboard in isolated Chrome.

Ask the operator to paste reads / likes / comments (and a note if they want). Then `studio_publish_mark` with those integers. Do not fetch the dashboard HTML. Do not click around the analytics UI.

If they want a postmortem sentence, write it into `note`. Leave topic scores untouched unless they ask to close the card (`status: posted` on the publish row already marks the topic done).
