---
name: studio-publish
description: Hand a shorts mp4 to a social creator page as a draft. Use when the operator asks to 发布, 投稿, 上传小红书/抖音/快手/视频号/知乎/B站. Yoyo never clicks Publish.
license: Apache-2.0
allowed-tools: studio_get studio_publish_draft studio_publish_list studio_publish_export studio_publish_open studio_publish_mark load_skill list_skills update_plan
---

# Studio publish (handoff, not autopost)

Yoyo does not scrape creator backends and does not click Publish. Playwright and SendAsYou are out of this lane.

1. `studio_get`. Need a `result_hash`. If missing, `load_skill` `studio-assemble`.
2. Confirm platform: `xhs` / `douyin` / `kuaishou` / `channels` / `zhihu` / `bilibili`.
3. `studio_publish_draft` with title (and caption/tags if the operator gave them). `project_id` is the bound short.
4. `studio_publish_open` — exports the mp4 next to the video home and opens the creator URL in isolated Chrome. The operator logs in, drops the file, and clicks Publish.
5. After they confirm, `studio_publish_mark` `posted` with `posted_url` if they have one. On failure, mark `failed` with a note.

Never invent a posted URL. Never call `browser_click` on a Publish button. For numbers after posting, `load_skill` `studio-analytics`.
