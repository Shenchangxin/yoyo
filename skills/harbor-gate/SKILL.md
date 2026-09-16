---
name: harbor-gate
description: Run Harbor evidence before promoting a harness. Use when the operator asks to checkout, promote, evolve, or land a snapshot. Not for ordinary code edits.
license: Apache-2.0
---

# Harbor gate

Promotion is evidence, not a chat decision.

1. Describe the candidate snapshot (hash, parent, note).
2. Tell the operator to run Harbor (held-in / held-out, optional TB) from the Harbor lab. Do not pretend the GUI eval is a sandbox.
3. Checkout only after Harbor says promote. L3 surfaces (`loop_preset`, `policy_pack`) require an explicit confirm.
4. Never instruct the agent loop to edit vault, updater, or evaluator config. Those are L4.
