import { useState } from "react";
import { ArrowUp } from "lucide-react";
import { useCopy } from "../../lib/i18n";
import { styleCoverFallback } from "../../vendor/yingce-canvas/lib/style-cover";
import { creationCases, type CreationCase, type CreationCaseLane } from "../../vendor/yingce-canvas/pages/create/creation-inspirations";

const PAGE = 12;

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

function laneLabel(lane: CreationCaseLane, c: ReturnType<typeof useCopy>["video"]["conductor"]) {
  if (lane === "studio") return c.featuredLaneStudio;
  if (lane === "drama") return c.featuredLaneDrama;
  if (lane === "canvas") return c.featuredLaneCanvas;
  return c.featuredLaneChat;
}

function sourceLine(item: CreationCase, c: ReturnType<typeof useCopy>["video"]["conductor"]) {
  const parts = [c.featuredStyle.replace("{title}", item.styleTitle).replace("{file}", item.coverFile)];
  if (item.sharedWith[0]) parts.push(c.featuredShared.replace("{title}", item.sharedWith[0]));
  if (!item.preset) parts.push(c.featuredStillOnly);
  return parts.join(" · ");
}

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
  const filtered = creationCases.filter((item) => filter === "all" || item.mode === filter);
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
        {filters.map((entry) => {
          const count = creationCases.filter((item) => entry.id === "all" || item.mode === entry.id).length;
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
          {shown.map((item, index) => (
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
                  onError={(event) => {
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
          <button type="button" className="creation-gallery-more" onClick={() => setLimit((count) => count + PAGE)}>
            {c.featuredMore}
          </button>
        ) : (
          <span>{c.featuredDone.replace("{n}", String(filtered.length))}</span>
        )}
      </footer>
    </section>
  );
}
