---
name: skill-creator
description: Create or update a Yoyo skill as a SKILL.md folder with name, description, and task-specific instructions. Use when the operator asks to write, split, or revise a skill.
license: Apache-2.0
metadata:
  short-description: Create or update a skill
---

# Skill creator

Write skills that change a later turn. Do not restock generic coding advice.

## Location

Prefer `workspace/.yoyo/skills/<name>/SKILL.md`. Home overlay is `$YOYO_HOME/skills`. Bundled skills live next to `evals/` and lose to workspace overlays.

## Format

```text
---
name: lowercase-hyphen-name
description: When to load this skill, including what it is not for.
---

# Title

Purpose, constraints, and the shortest procedure that actually changes the work.
```

`name` and `description` are required. Description is routing: say when to load it and when not to.

## Rules

- Keep the body short. Put mode-specific schemas in `references/` and read them only when needed.
- Use `scripts/` only for deterministic helpers the model would otherwise rewrite.
- Do not grant extra capabilities. Skills cannot bypass the gate, vault, Harbor, or updater.
- After writing, call `list_skills` and `load_skill` to confirm the name resolves.

## Distill a trajectory

When asked to turn a successful turn into an Expert:

- Expert = `SKILL.md` + optional `EVALS.txt` + Harbor suite ids + policy. Not a persona pack.
- Distill only the procedure that changed the work. Drop chatter.
- Unsigned packs cannot become `refs/active`. Write the skill under `$YOYO_HOME/skills/<name>/` as staging; admit requires an ed25519 `SKILL.sig`.
