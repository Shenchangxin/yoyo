---
name: drama-prompt-prop
description: Write still prompts for Short drama hero props on a white seamless. Use when the operator starts the prompt stage for props.
license: Apache-2.0
allowed-tools: drama_read_assets drama_save_final_prompt load_skill list_skills update_plan
---

# Prop still prompt

Read assets. For each linked prop without a strong `final_prompt`, write a product still: silhouette, materials, wear. White seamless. No hands. No room.

Do not mention the project style; `drama_save_final_prompt` prefixes it. kind=`prop`. Do not generate images.
