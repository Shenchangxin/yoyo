---
name: plan-first
description: Keep a visible task plan with update_plan. Use for multi-step work, refactors, and anything the operator marked Plan. Not for a single-file one-liner.
license: Apache-2.0
---

# Plan first

Call `update_plan` before the first write.

- Steps are short. Status is `pending`, `in_progress`, or `complete`.
- Exactly one step `in_progress` at a time.
- Plan mode cannot write or shell; propose, then wait.
- Re-run `update_plan` when the step changes. Do not narrate the plan in prose instead of the tool.
