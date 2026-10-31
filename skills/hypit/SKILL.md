---
name: hypit
description: Finish a commissioned film as a watchable short. Use when the operator hands over a talking, drama, podcast, interview, ranking, explainer, or narration brief, or mentions @skill:hypit.
license: Apache-2.0
allowed-tools: video_route provider_menu studio_get studio_update studio_apply_template studio_approve_script studio_preflight studio_generate_missing studio_speak studio_apply_timing studio_assemble ask_user update_plan load_skill
---

# Hypit

This skill ships with Yoyo. The operator talks to you; you make the short with the shorts tools that ship in the app.

The creation desk already collected the brief. Keep their subject, people, and lines. `provider_menu` before image or speech. Missing Settings accounts do not exist. `ask_user` before `studio_approve_script`.

## Open the short

`video_route` dest=studio, title = the subject. `studio_*` attaches once that short is bound. If those tools are absent on this turn, stop after the route and continue on the bound short.

`studio_apply_template`:

- talking, explainer, narration: `explainer`
- ranking, interview: `howto`
- podcast: `ugc-hook`
- drama: `story`

## Write what they will hear

Spoken text only, in the operator's language. One shot per beat. Keep the relationship the brief names:

- talking: one speaker stays, and each row is the next line
- drama: both people stay in the scene; the ask, then the reply
- podcast: the question stays when the reply arrives
- interview: answers appear one at a time
- ranking: earlier places stay on the board as later places arrive
- explainer: the point stays while the picture changes
- narration: the voice stays and the picture changes

`studio_update` with `script`, `hook`, and `shots`. Each shot has a `caption`, a `visual` prompt, `duration` of 2–5 seconds, and `motion` `ken-burns`. Show the spoken lines, then `studio_approve_script` only after they confirm.

## Picture, voice, file

`studio_preflight` `visuals`. `studio_generate_missing` for stills. `studio_speak` the approved script, `studio_apply_timing`, then `studio_assemble`.

Report the result hash. The finished film is that mp4 in the shorts library. Leave hashes and keys as the tools return them.
