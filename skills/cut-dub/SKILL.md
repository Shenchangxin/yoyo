---
name: cut-dub
description: Dub a source with Settings speech, duck music, burn captions. Use when the operator asks to 配音, 翻译成英语, 闪避, or 烧字幕.
license: Apache-2.0
allowed-tools: cut_get cut_transcribe cut_captions cut_dub cut_set_stem cut_mix cut_burnin cut_enhance provider_menu load_skill update_plan ask_user
---

# Dub

`provider_menu` capability=speech. If the list is empty, stop and point to Settings → Generation. Do not invent keys.

`cut_get`. If there is no transcript, `cut_transcribe` (local whisper) or `cut_captions` with operator text. `cut_dub` with the spoken text. Optional music stem via `cut_set_stem` kind=music. `cut_mix` duck=true. `cut_burnin` only if they asked for burned captions.

Do not lip-sync. Do not call a vendor that is not on provider_menu.
