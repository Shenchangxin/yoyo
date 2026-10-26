/**
 * 本文件拥有创作案例画廊的筛选和翻页。不拥有案例数据。
 * 点卡片只把文案交给回调，本文件不改服务端。
 */
import { useState } from "react";
import { ArrowUp } from "lucide-react";
import { useCopy } from "../../lib/i18n";
import { styleCoverFallback } from "../../vendor/yingce-canvas/lib/style-cover";
import { creationCases, type CreationCase, type CreationCaseLane } from "../../vendor/yingce-canvas/pages/create/creation-inspirations";

const PAGE = 12;

/** 按案例通道套上标题和提示词。item 按 lane 选模板，再把 title 和 prompt 填进去。c 提供各通道的填充文案。 */
function fillFor(item: CreationCase, c: ReturnType<typeof useCopy>["video"]["conductor"]) {
  const text = item.lane === "studio"
    ? c.featuredFillStudio
    : item.lane === "drama"
      ? c.featuredFillDrama
      : item.lane === "canvas"
        ? c.featuredFillCanvas
        : item.prompt;
  return text.replace("{title}", item.title).replace("{prompt}", item.prompt);
}

/** 把案例通道换成界面上的名字。lane 换成 studio、drama、canvas 或聊天的名字。c 提供这些名字。 */
function laneLabel(lane: CreationCaseLane, c: ReturnType<typeof useCopy>["video"]["conductor"]) {
  if (lane === "studio") return c.featuredLaneStudio;
  if (lane === "drama") return c.featuredLaneDrama;
  if (lane === "canvas") return c.featuredLaneCanvas;
  return c.featuredLaneChat;
}

/** 拼风格、共用对象，以及没有预设时的静帧说明。item 用 styleTitle、coverFile、sharedWith 和 preset 拼一行。c 提供风格、共用和静帧文案。 */
function sourceLine(item: CreationCase, c: ReturnType<typeof useCopy>["video"]["conductor"]) {
  const parts = [c.featuredStyle.replace("{title}", item.styleTitle).replace("{file}", item.coverFile)];
  if (item.sharedWith[0]) parts.push(c.featuredShared.replace("{title}", item.sharedWith[0]));
  if (!item.preset) parts.push(c.featuredStillOnly);
  return parts.join(" · ");
}

/** FeaturedCases 每次再多展示 12 条案例。封面失败时换备用图。props 的 onPrompt 在点卡片时收到填好的文案。 */
export function FeaturedCases(props: { onPrompt: (text: string) => void }) {
  const copy = useCopy();
  const c = copy.video.conductor;
  const [filter, setFilter] = useState<"all" | CreationCase["mode"]>("all");
  const [limit, setLimit] = useState(PAGE);
  const filters = [
    { id: "all" as const, label: c.featuredAll },
    { id: "video" as const, label: c.featuredVideo },
    { id: "image" as const, label: c.featuredImage },
    { id: "text" as const, label: c.featuredText },
  ];
  const filtered = creationCases.filter(/** item 在筛选为 all，或 mode 和筛选相同时留下。 */ (item) => filter === "all" || item.mode === filter);
  const shown = filtered.slice(0, limit);
  return (
    <section className="creation-gallery" data-testid="creation-cases" aria-labelledby="creation-cases-title">
      <div className="creation-gallery-head">
        <h2 id="creation-cases-title">{c.featured}</h2>
        <p>{c.featuredHint.replace("{n}", String(creationCases.length))}</p>
      </div>
      <p className="creation-gallery-source" data-testid="creation-cases-source">
        {c.featuredSource.replace("{n}", String(creationCases.length))}
      </p>
      <div className="creation-gallery-filters" role="group" aria-label={c.featured}>
        {filters.map(/** entry 用 id 统计数量和选中态，用 label 显示。 */ (entry) => {
          const count = creationCases.filter(/** item 在 entry 为 all，或 mode 和 entry.id 相同时计入。 */ (item) => entry.id === "all" || item.mode === entry.id).length;
          return (
            <button
              key={entry.id}
              type="button"
              className="creation-gallery-filter"
              aria-pressed={filter === entry.id}
              onClick={() => { setFilter(entry.id); setLimit(PAGE); }}
            >
              {entry.label}
              <span>{count}</span>
            </button>
          );
        })}
      </div>
      {shown.length === 0 ? <p className="creation-gallery-empty">{c.featuredEmpty}</p> : (
        <div className="creation-gallery-grid">
          {shown.map(/** item 画封面、标题和说明。index 为 0 且这条是精选、筛选为 all 时加大卡片。 */ (item, index) => (
            <button
              key={item.id}
              type="button"
              className={filter === "all" && index === 0 && item.featured ? "creation-gallery-card is-hero" : "creation-gallery-card"}
              data-testid={`creation-case-${item.id}`}
              aria-label={`${item.title}. ${laneLabel(item.lane, c)}. ${sourceLine(item, c)}`}
              onClick={() => props.onPrompt(fillFor(item, c))}
            >
              <span className="creation-gallery-media">
                <img
                  src={item.image}
                  alt=""
                  loading="lazy"
                  onError={/** event 的 currentTarget 在还没换过备用图时改 src。 */ (event) => {
                    const img = event.currentTarget;
                    if (img.dataset.fallback === "1") return;
                    img.dataset.fallback = "1";
                    img.src = styleCoverFallback(item.image);
                  }}
                />
                <span className="creation-gallery-use"><ArrowUp className="size-3" aria-hidden />{c.featuredUse}</span>
              </span>
              <span className="creation-gallery-copy">
                <em>{laneLabel(item.lane, c)}</em>
                <strong>{item.title}</strong>
                <span>{item.description}</span>
                <small>{sourceLine(item, c)}</small>
              </span>
            </button>
          ))}
        </div>
      )}
      <footer className="creation-gallery-foot">
        {limit < filtered.length ? (
          <button type="button" className="creation-gallery-more" onClick={() => setLimit(/** count 加上 12。 */ (count) => count + PAGE)}>
            {c.featuredMore}
          </button>
        ) : (
          <span>{c.featuredDone.replace("{n}", String(filtered.length))}</span>
        )}
      </footer>
    </section>
  );
}
