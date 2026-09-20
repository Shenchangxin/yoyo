# 文献来源渠道

审阅：**2026-09-20**。本文件是检索协议。论文卡片仍按一篇一卡进 [corpus/](corpus/) 或 [products/](products/)，id 进 [catalog.yaml](catalog.yaml)。

年份门不变：前沿只收 2026-first。会场接收但 arXiv 首发在 2025 的（ACE、HGM）进谱系。

## 渠道表

| 渠道 | 角色 | 怎么用 | 原文怎么存 |
| --- | --- | --- | --- |
| **arXiv** | 主库 | `cs.AI` / `cs.CL` / `cs.SE` / `cs.LG`；query 含 harness, RSI, agent, Terminal-Bench | `scripts/fetch-research-originals.ps1` → `originals/<id>.pdf` |
| **OpenReview** | 会场审稿与 workshop | ICLR / ICML / NeurIPS / ACL 的 forum、被拒意见、camera-ready | 链到 forum；有 arXiv 的仍拉 PDF |
| **ICLR / ICML / ACL / EMNLP / NeurIPS** | 正式发表 | 2026 主会 + workshop。NeurIPS 2026、EMNLP 2026 在今天（9 月）多为投稿中，刷新时再扫 | 会场 PDF 或 arXiv |
| **Hugging Face Daily Papers** | 发现 | 按 arXiv id 的当日榜，补手搜漏网 | 不缓存页面；跟 id 回 arXiv |
| **alphaXiv** | 发现 + 讨论 | 摘要页、社区笔记 | 同上 |
| **Semantic Scholar / Connected Papers** | 引用图 | 从 Self-Harness、ModularRSI、AHE、Meta-Harness 向外扩 1-hop | 只当发现，不进 catalog 当文献 |
| **厂商工程博客** | 生产 harness | OpenAI、Anthropic、Google、xAI、Cursor、Harbor | `products/` 卡片 + 官方 URL（不把付费/动态 HTML 塞 git） |
| **配套 GitHub** | 可运行证据 | 论文卡上的 `代码` 栏 | 不镜像整个上游仓 |

不要当主源：IEEE/ACM 付费全文、无官方 URL 的 slidedeck、社交媒体 thread。

## 固定入口

```
arXiv API          https://export.arxiv.org/api/query?search_query=...
arXiv abs          https://arxiv.org/abs/<id>
HF Papers          https://huggingface.co/papers/<id>
alphaXiv           https://www.alphaxiv.org/abs/<id>
OpenReview 搜索    https://openreview.net/search
ICLR 2026 RSI WS   https://iclr.cc/virtual/2026/workshop/10000796
                   OpenReview venue: ICLR.cc/2026/Workshop/RSI
ICML 2026          https://icml.cc/virtual/2026
OpenAI             https://openai.com/index/  + https://openai.com/research/
Anthropic eng      https://www.anthropic.com/engineering
Google Dev Blog    https://developers.googleblog.com/
xAI                https://x.ai/news
Cursor             https://cursor.com/blog  + Composer 报告走 arXiv
Harbor             https://www.harborframework.com  + https://harbor-index.org
Semantic Scholar   https://www.semanticscholar.org/search?q=
Connected Papers   https://www.connectedpapers.com/
```

OpenReview 已对齐的 forum（有 arXiv 的仍以 arXiv 为原文）：

| 工作 | OpenReview |
| --- | --- |
| ACE | https://openreview.net/forum?id=f1eKYFvMNu |
| AutoHarness | https://openreview.net/forum?id=g9rEYVNn5T |
| Reward Hacking in Self-Improving Code Agents | https://openreview.net/forum?id=ikrQWGgxYg |
| ICLR 2026 RSI workshop 列表 | https://openreview.net/submissions?venue=ICLR.cc/2026/Workshop/RSI |
| ICML 2026 Meta-Harness | https://icml.cc/virtual/2026/67972 |

## 每季必查（约 30 分钟）

1. arXiv：`ti:"harness" AND (agent OR RSI OR self-improv)`，日期 ≥ 上次 `last_reviewed`。
2. HF Daily Papers + alphaXiv：同一 query，抓新 id。
3. OpenReview：ICLR/ICML 当年 + RSI / Agent / Workshop 组。
4. 会场虚拟站：ICLR、ICML；9 月之后加 EMNLP、NeurIPS。
5. 博客：Anthropic engineering、OpenAI index、Google Developers、x.ai/news、Cursor blog。
6. Connected Papers：以 `2606.09498`、`2609.14857`、`2604.25850`、`2603.28052` 为种子，1-hop 里 year=2026 的新点。
7. 新 id → `catalog.yaml` → 卡片 → 更新 [FRONTIER.md](FRONTIER.md) → 手写更新 [report.html](report.html) → 跑 fetch 脚本。

ACL/EMNLP 2026 主会截至本次审阅 **没有** 与 Yoyo 控制塔直接对齐的已发表 harness/RSI 论文；下次刷新继续扫 ARR。NeurIPS 2026 尚未开会。
