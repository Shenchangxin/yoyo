# Search

`web_search` is the only generic search tool.

Order inside the host (you do not pick this):

1. Configured Search connection (Tavily / Brave / Serper / Bocha / Metaso / Exa) when it returns hits
2. Jina Search (no key)
3. DuckDuckGo HTML, parsed into title/url/snippet

If results look thin, change the query. Do not retry via curl. Do not call `reach_status` in a loop.

Chinese product/news queries: still `web_search`. A Bocha/Metaso key in Settings improves this; say so only if results are unusable.

After search, `web_fetch` the two or three URLs you will cite. Do not dump SERP chrome into the deliverable.
