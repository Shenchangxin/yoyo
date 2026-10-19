import { useState } from "react";
import { ArrowUp } from "lucide-react";
import { creationFeaturedWorks } from "@yingce/pages/create/creation-inspirations";
import { styleCoverFallback, styleCoverSrc } from "@yingce/lib/style-cover";
import { useCopy } from "../../lib/i18n";
import { cn } from "../../lib/utils";

type Mode = "all" | "video" | "image" | "text";

export function CreationGallery(props: { onStartPrompt: (prompt: string) => void }) {
  const copy = useCopy();
  const c = copy.video.conductor;
  const [filter, setFilter] = useState<Mode>("all");
  const [limit, setLimit] = useState(12);
  const filtered = creationFeaturedWorks.filter((item) => filter === "all" || item.mode === filter);
  const shown = filtered.slice(0, limit);
  const modeLabel: Record<Exclude<Mode, "all">, string> = {
    video: c.featuredVideo,
    image: c.featuredImage,
    text: c.featuredText,
  };
  const filters: { id: Mode; label: string; count: number }[] = [
    { id: "all", label: c.featuredAll, count: creationFeaturedWorks.length },
    { id: "video", label: c.featuredVideo, count: creationFeaturedWorks.filter((it) => it.mode === "video").length },
    { id: "image", label: c.featuredImage, count: creationFeaturedWorks.filter((it) => it.mode === "image").length },
    { id: "text", label: c.featuredText, count: creationFeaturedWorks.filter((it) => it.mode === "text").length },
  ];

  return (
    <section className="creation-gallery" data-testid="creation-featured" aria-labelledby="creation-featured-title">
      <header className="creation-gallery-head">
        <div>
          <h2 id="creation-featured-title" className="text-[15px] font-semibold tracking-[-0.03em]">
            {c.featured}
          </h2>
          <p className="mt-1 text-[12px] leading-[1.5] text-muted">
            {c.featuredHint.replace("{n}", String(filtered.length))}
          </p>
        </div>
      </header>
      <div className="creation-gallery-filters" role="group" aria-label={c.featured}>
        {filters.map((item) => (
          <button
            key={item.id}
            type="button"
            className="creation-gallery-filter"
            aria-pressed={filter === item.id}
            disabled={item.count === 0}
            onClick={() => {
              setFilter(item.id);
              setLimit(12);
            }}
          >
            {item.label}
            <span>{item.count}</span>
          </button>
        ))}
      </div>
      {shown.length === 0 ? (
        <p className="creation-gallery-empty">{c.featuredEmpty}</p>
      ) : (
        <div className="creation-gallery-grid">
          {shown.map((item, index) => {
            const hero = filter === "all" && index === 0 && !!item.featured;
            return (
              <button
                key={`${item.mode}-${item.title}-${index}`}
                type="button"
                className={cn("creation-gallery-card", hero && "is-hero")}
                aria-label={`${item.title} · ${modeLabel[item.mode]} · ${c.featuredUse}`}
                onClick={() => props.onStartPrompt(item.prompt)}
              >
                <span className="creation-gallery-media">
                  <img
                    src={styleCoverSrc(item.image)}
                    alt=""
                    loading="lazy"
                    onError={(e) => {
                      e.currentTarget.src = styleCoverFallback(item.image);
                    }}
                  />
                  <span className="creation-gallery-use">
                    <ArrowUp className="size-3.5" aria-hidden />
                    {c.featuredUse}
                  </span>
                </span>
                <span className="creation-gallery-copy">
                  <em>{modeLabel[item.mode]}</em>
                  <strong>{item.title}</strong>
                  <span>{item.description}</span>
                </span>
              </button>
            );
          })}
        </div>
      )}
      <footer className="creation-gallery-foot">
        {limit < filtered.length ? (
          <button type="button" className="creation-gallery-more" onClick={() => setLimit((n) => n + 12)}>
            {c.featuredMore}
          </button>
        ) : (
          <span>{c.featuredDone.replace("{n}", String(filtered.length))}</span>
        )}
        <p>{c.featuredNotice}</p>
      </footer>
    </section>
  );
}
