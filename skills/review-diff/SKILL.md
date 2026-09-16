---
name: review-diff
description: Review git hunks in the current workspace before apply. Use when the operator asks to inspect, stage, or apply a patch, not for Harbor promotion.
license: Apache-2.0
---

# Review diff

The Review pane is the apply surface. Do not dump the whole repo.

1. Call `git_diff` (and `git_status` if the working tree is unclear).
2. Summarize files and risk. Call out secrets, generated lockfiles, and harness files under `loop_preset` / `policy_pack`.
3. Prefer `apply_patch` for surgical edits. Leave Apply selected to the operator when hunks are already in Review.
4. Stop after the review unless the operator asked you to write.
