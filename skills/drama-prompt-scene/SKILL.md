---
name: drama-prompt-scene
description: Write still prompts for Short drama locations as empty establishing wides. Use when the operator starts the prompt stage for scenes.
license: Apache-2.0
allowed-tools: drama_read_assets drama_save_final_prompt load_skill list_skills update_plan
---

# Scene still prompt

Read assets. For each linked scene without a strong `final_prompt`, write an establishing wide: architecture, time of day, weather, practical light. No people. No named characters.

Do not mention the project style; `drama_save_final_prompt` prefixes it. kind=`scene`. Do not generate images.
