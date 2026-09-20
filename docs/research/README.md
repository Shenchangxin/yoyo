# Yoyo research corpus

Yoyo 的产品护城河是 **可版本化、可评测、可晋升、可回滚的 harness**。本目录把这件事放到学术与工业报告的坐标系里，方便 owner 按证据迭代，而不是按感觉加 prompt。

最后审阅：**2026-09-20**。前沿子库只收录 **2026 年首次出现或 2026 年会场发表** 的材料。2025 年工作一律放进 [REFERENCES.md](REFERENCES.md) 的「谱系」层，不当成当前前沿。检索渠道见 [SOURCES.md](SOURCES.md)。

**Owner 入口：** 打开 [report.html](report.html)。手写扫读件，不是脚本生成。Markdown 卡片仍是源。改文献必须同轮改报告，见维护协议第 8 条。

## 读什么

| 文档 | 用途 |
| --- | --- |
| [report.html](report.html) | **一篇简报**：判断、下一步、不做、文献。从上往下读。 |
| [REFERENCES.md](REFERENCES.md) | **本项目已经引用 / 已经落地** 的文献与产品技术报告：核心思想、对应代码、我们刻意没做的部分 |
| [FRONTIER.md](FRONTIER.md) | **2026 最有价值 / 最前沿 / 最前瞻** 的论文与报告体系：排序、与 Yoyo 的差距、下一批该做什么 |
| [catalog.yaml](catalog.yaml) | 机器可读目录（id、日期、状态、Yoyo 映射）。新增条目先改这里 |
| [corpus/](corpus/) | 单篇摘要卡片（我们的话，不是全文转写） |
| [products/](products/) | 产品技术报告 / 官方工程说明 |
| [SOURCES.md](SOURCES.md) | 来源渠道与每季检索协议（arXiv 之外的 OpenReview / 会场 / 厂商博客 / HF / alphaXiv / 引用图） |
| [originals/](originals/) | arXiv PDF 本地缓存。默认不进 git，用脚本拉取 |

## 两条轴

1. **谱系（what we built on）** — ACE 式 playbook、Self-Harness 闭环、DGM 档案采样、Harbor 评测布局、Codex 式 App Server、Claude 式 skills / 权限。见 REFERENCES。
2. **前沿（what we should steal or reject next）** — ModularRSI 的模块化与 benchmark-disjoint、HarnessEvolve 的参考轨迹与双门、Wang / Gideoni 对「进化是否只是多花了搜索预算」的质疑、Harbor-Index、技能供应链、ICLR RSI / ICML 2026 新工作。见 FRONTIER。

Self-harness 与 RSI 在本仓库里不是口号：`yoyo evolve` 是 L1 闭环；Harbor 拥有 `refs/active`；Go 内核是 L4，智能体改不了。

## 维护协议

Owner 或后续 agent 更新本目录时遵守：

1. **年份门** — `catalog.yaml` 的 `year_first` 必须是 2026（谱系条目用 `layer: lineage` 并写清 2025 首发日期）。过了 2027-01-01，把 2026 年工作降为谱系，重新做一轮 2027 检索。
2. **一篇一卡** — `corpus/<arxiv-id>-<slug>.md`。卡片写核心主张、对 Yoyo 的含义、可偷 / 应拒，不粘贴论文正文。
3. **原文走官方渠道** — 有 arXiv id 的 PDF 用 `scripts/fetch-research-originals.ps1` 拉取。OpenReview / 厂商博客只链官方 URL。不要把付费期刊全文塞进 git。
4. **多渠道检索** — 按 [SOURCES.md](SOURCES.md) 扫 arXiv、OpenReview、会场虚拟站、HF Daily / alphaXiv、厂商博客、Semantic Scholar / Connected Papers。发现用 HF/alphaXiv，归档以 arXiv 或官方博客为准。
5. **映射必须落到代码** — REFERENCES 里每一条「我们参考了」都要有 `internal/...` 路径。做不到就标 `not-implemented`。
6. **状态词汇** — `implemented` | `partial` | `candidate` | `critique` | `watch` | `lineage`。
7. **版权** — 摘要用我们的话。官方 abstract 只保留一两句并标明出处。需要全文时打开 `originals/` 或 arXiv HTML。
8. **HTML 报告** — `report.html` 是手写的 owner 扫读件。改 catalog / 卡片 / FRONTIER / REFERENCES 的同一轮必须改报告。禁止用脚本从 Markdown 生成或覆盖它。契约：`.cursor/rules/research-report.mdc`。

```powershell
# 从仓库根目录拉取 PDF 缓存
powershell -File scripts/fetch-research-originals.ps1
```
