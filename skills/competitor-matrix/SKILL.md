---
name: competitor-matrix
description: Build a competitor comparison as a formula spreadsheet plus a cited markdown report. Use for 竞品 / competitor matrix research.
license: Apache-2.0
---

# Competitor matrix

1. `web_search` then `web_fetch` primary sources. Isolated browser only if fetch is not enough.
2. `office_create` matrix.xlsx with named vendors as rows and scored columns plus a SUM or AVERAGE formula.
3. Write report.md and `cite_sources` with URL + excerpt + hash. Hallucinated URLs fail Harbor.
4. Fan out `task` with isolate=true for parallel vendor pages. Depth stays 1.
