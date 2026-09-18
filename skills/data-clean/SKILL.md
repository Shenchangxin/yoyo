---
name: data-clean
description: Clean a table into a real .xlsx with formulas, not a CSV that looks like a sheet. Use for data cleaning / 数据清洗.
license: Apache-2.0
---

# Data clean

1. Read the source file. Keep a copy.
2. `office_create` cleaned.xlsx. Numeric columns stay numbers. Totals use `=SUM`.
3. `office_query` / `office_render` to verify evaluated totals.
4. `fs_batch` only inside the workspace jail.
