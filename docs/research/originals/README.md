# Originals

arXiv PDF 缓存。**不提交 git**（体积 + 各篇许可证不同）。需要全文时从这里打开，或用官方 HTML。

```powershell
# repo root: yoyo/
powershell -File scripts/fetch-research-originals.ps1
```

文件名：`<arxiv-id>.pdf`，与 [catalog.yaml](../catalog.yaml) 的 `id` 对应。

不要把出版社付费 PDF、被版权保护的书籍章节放进这个目录。
