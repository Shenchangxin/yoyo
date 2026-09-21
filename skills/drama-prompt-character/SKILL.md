---
name: drama-prompt-character
description: Write still prompts for Short drama characters. Left a front portrait, right a three-view turnaround, empty background. Use when the operator starts the prompt stage for the cast.
license: Apache-2.0
allowed-tools: drama_read_assets drama_save_final_prompt load_skill list_skills update_plan
---

# Character still prompt

Read assets. For each linked character without a strong `final_prompt`, write one visual description: face, body, costume, materials. Left panel is a tight front portrait. Right panel is a three-view turnaround. Empty background.

Do not mention the project style; `drama_save_final_prompt` prefixes it. kind=`character`. Do not generate images.
