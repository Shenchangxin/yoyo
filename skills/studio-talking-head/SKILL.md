---
name: studio-talking-head
description: Burn a title or script onto a CAS video. Default talking-head wrap. Use for 口播包装, 标题烧录, talking-head. Remotion is operator-installed and never in the default installer.
license: Apache-2.0
allowed-tools: studio_get studio_wrap studio_burnin studio_remotion_status studio_remotion_install load_skill list_skills update_plan
---

# Talking-head wrap

`studio_get` for `result_hash`, or take a hash the operator pasted. `studio_wrap` with that hash and the title (or a tight on-screen line from the script).

`studio_wrap` uses Remotion TitleCard when `studio_remotion_status` shows `ready`; otherwise FFmpeg SRT burn-in. Report the new CAS hash. Do not invent hashes. If FFmpeg is missing, point to Settings → Generation.

If they want Remotion and it is not ready, `load_skill` `studio-remotion` and offer install. Do not add Remotion to the default installer.
