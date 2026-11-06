/**
 * 拥有：生成页上面那排类别。点一个类别只留下对应的服务块；深链到某一块时自动切到它所在的类别。
 * 不拥有：各服务的表单，那些在 GenerationSettings。显示和隐藏靠 styles.css 里的 data-gen-filter 规则。
 */
import { useEffect, useState, type ReactNode } from "react";
import { useCopy } from "../../lib/i18n";
import { useUI } from "../../lib/store";
import { cn } from "../../lib/utils";

type Category = "all" | "studio" | "image" | "video" | "speech" | "music" | "matting" | "stock" | "asr" | "more";

const SECTION_CATEGORY: Record<string, Category> = {
  "generation-studio": "studio",
  "generation-image": "image",
  "generation-video": "video",
  "generation-speech": "speech",
  "generation-music": "music",
  "generation-matting": "matting",
  "generation-stock": "stock",
  "generation-asr": "asr",
  "generation-storage": "more",
  "generation-search": "more",
  "generation-workflow": "more",
  "generation-canvas": "more",
};

/** GenerationCatalog 包住生成页，顶上放类别筛选。 */
export function GenerationCatalog({ children }: { children: ReactNode }) {
  const copy = useCopy();
  const g = copy.settings.catalog;
  const section = useUI((s) => s.settingsSection);
  const nav = useUI((s) => s.settingsNav);
  const [filter, setFilter] = useState<Category>("all");
  useEffect(() => {
    const hit = SECTION_CATEGORY[section];
    if (hit) setFilter(hit);
  }, [section, nav]);
  const items: { id: Category; label: string }[] = [
    { id: "all", label: g.all },
    { id: "studio", label: g.studio },
    { id: "image", label: g.image },
    { id: "video", label: g.video },
    { id: "speech", label: g.speech },
    { id: "music", label: g.music },
    { id: "matting", label: g.matting },
    { id: "stock", label: g.stock },
    { id: "asr", label: g.asr },
    { id: "more", label: g.more },
  ];
  return (
    <div data-gen-filter={filter}>
      <div role="tablist" aria-label={copy.settings.sections.generationCatalog} className="mb-5 flex flex-wrap gap-1.5" data-testid="generation-catalog">
        {items.map((item) => {
          const on = filter === item.id;
          return (
            <button
              key={item.id}
              type="button"
              role="tab"
              aria-selected={on}
              className={cn(
                "h-7 rounded-full border px-3 text-ui-footnote transition-colors duration-[var(--duration-fast)]",
                on ? "border-foreground/20 bg-lift font-medium text-foreground" : "border-border text-muted hover:text-foreground",
              )}
              onClick={() => setFilter(item.id)}
            >
              {item.label}
            </button>
          );
        })}
      </div>
      {children}
    </div>
  );
}
