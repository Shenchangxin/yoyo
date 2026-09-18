---
name: weekly-report
description: Produce a weekly status report as a real .docx plus a formula .xlsx. Use when the operator asks for a 周报, weekly update, or status pack.
license: Apache-2.0
---

# Weekly report

Deliver artifacts, not chat.

1. Read the workspace and project memory for facts only.
2. `office_create` a `.docx` with title, this week, risks, next week.
3. `office_create` a `.xlsx` with numeric rows and at least one `=SUM(...)` formula.
4. `office_render` and fix empty sections.
5. Never send mail. Put the draft in the workspace and `notify_actionable`.
