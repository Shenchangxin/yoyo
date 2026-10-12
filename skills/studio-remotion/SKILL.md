---
name: studio-remotion
description: Install or use Remotion for shorts title-card wrap. Use when they ask for Remotion, React video, or a talking-head title overlay. Never add Remotion to the default installer.
license: Apache-2.0
allowed-tools: studio_remotion_status studio_remotion_install studio_wrap studio_get load_skill list_skills update_plan
---

# Remotion (opt-in)

`studio_remotion_status`. `bundled` is always false. The default installer does not ship Node, Remotion, or ~540 MB of extras.

- If `ready` is true, the managed composition under the video home can render. `studio_wrap` uses Remotion TitleCard, then falls back to FFmpeg if render fails.
- If `ready` is false and they ask to install: confirm, then `studio_remotion_install` (needs Node.js 18+). Poll `studio_remotion_status` until `ready` or `install.phase` is `error`. Do not run `npx remotion` to download a random package. Do not clone video-pipeline-sdk.
- If they decline install, `load_skill` `studio-talking-head` (FFmpeg wrap).
