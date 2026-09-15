---
name: component-architecture
description: Use when structuring React components — composing shadcn/ui, cva variants, compound components, controlled vs uncontrolled state, and knowing when to abstract versus inline.
---

# Component Architecture

```
frontend/src/
  components/ui/     # shadcn primitives only
  features/          # agent, inspector, labs
  lib/               # client, protocol, stream, cn
```

- Variants via `cva`, not boolean soup.
- Wrap primitives; do not stuff app logic into `components/ui`.
- Rule of three before extracting.
- `App.tsx` must not own labs + composer + diff. Split stores and feature roots.
- Uncontrolled by default; control when the parent must sync (active thread id).
