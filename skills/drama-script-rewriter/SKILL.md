---
name: drama-script-rewriter
description: Rewrite a novel chapter into a formatted screenplay for Short drama. Use when the operator starts the script stage or asks to turn prose into shots and dialogue.
license: Apache-2.0
allowed-tools: drama_read_episode drama_save_script load_skill list_skills update_plan
---

# Screenplay rewrite

The episode is already bound. Call `drama_read_episode` first.

Write a screenplay, not a summary.

- Scene headings: location, interior/exterior, time of day.
- Action lines in present tense, visible and audible only.
- Character cues in a consistent name that later extraction can match.
- Dialogue on its own lines. Parentheticals only when the performance is not obvious.
- Keep the plot, relationships, and spoken information. Cut narrator commentary that cannot be filmed.
- Target length follows the duration plan implied by the source (about 500 characters per minute of finished clip).

Then call `drama_save_script` with the full screenplay. Do not narrate. Do not generate images or video.
