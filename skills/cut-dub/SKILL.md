---
name: cut-dub
description: Dub a source with Settings speech, duck music, burn captions. Use when the operator asks to 配音, 翻译成英语, 闪避, or 烧字幕.
license: Apache-2.0
allowed-tools: cut_get cut_transcribe cut_captions cut_dub cut_set_stem cut_mix cut_burnin cut_enhance provider_menu load_skill update_plan ask_user
---

# Dub

`provider_menu` capability=speech. If the list is empty, local edge-tts may still speak Chinese; say so, and point to Settings → Generation for other voices. Do not invent keys.

`cut_get`. If there is no transcript, `cut_transcribe` (local whisper.cpp with a model) or `cut_captions` with operator text. `cut_dub` with the spoken text; it only writes the dub stem. Optional music stem via `cut_set_stem` kind=music. Then `cut_mix` duck=true: the original audio stays under the dub. Pass replace=true only when the dub should replace the original voice, such as a translation. The film keeps its picture length. `cut_burnin` only if they asked for burned captions.

Do not lip-sync. Do not call a vendor that is not on provider_menu.
