---
name: forms-and-validation
description: Use when building forms with react-hook-form + zod and shadcn Form — schema validation, inline errors, accessible fields, and async submit.
---

# Forms & Validation

Stack: **react-hook-form + zod + shadcn Form**. Do not `useState` per Control field.

- One schema → types + validation. Reuse mentally against Go config, even if the wire is Wails.
- `defaultValues` always. `autoComplete` / `type` set.
- Disable submit only while `isSubmitting`. Toast on success. `setError` for server/Wails errors.
- API key field: never echo; `autoComplete="off"`; write on explicit save, not only `onBlur`.
- Unsaved-changes bar on Control.
