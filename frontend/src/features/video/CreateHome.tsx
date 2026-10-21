import { useEffect, useState } from "react";
import { Clapperboard, Film, Images, PanelsTopLeft, Scissors } from "lucide-react";
import * as api from "../../lib/client";
import { useCopy } from "../../lib/i18n";
import type { CutProject } from "./cut-desk";

function asCuts(v: unknown): CutProject[] {
  return Array.isArray(v) ? v as CutProject[] : [];
}

export function CreateHome(props: {
  onImport: () => void;
  onHandoff: (dest: "studio" | "drama" | "canvas") => void;
  onOpenCut: (id: string) => void;
}) {
  const copy = useCopy();
  const c = copy.video.conductor;
  const [cuts, setCuts] = useState<CutProject[]>([]);

  useEffect(() => {
    let alive = true;
    void api.video.cutList().then((list) => {
      if (!alive) return;
      const rows = asCuts(list).filter((row) => row?.id && row.source_hash);
      rows.sort((a, b) => String(b.updated_at || "").localeCompare(String(a.updated_at || "")));
      setCuts(rows.slice(0, 8));
    }).catch(() => {
      if (alive) setCuts([]);
    });
    return () => { alive = false; };
  }, []);

  const cards: { id: "studio" | "drama" | "canvas" | "continue"; title: string; hint: string; icon: typeof Film }[] = [
    { id: "studio", title: c.handoffShort, hint: c.handoffShortHint, icon: Film },
    { id: "drama", title: c.handoffDrama, hint: c.handoffDramaHint, icon: Clapperboard },
    { id: "canvas", title: c.handoffCanvas, hint: c.handoffCanvasHint, icon: PanelsTopLeft },
    { id: "continue", title: c.handoffContinue, hint: cuts[0]?.title || c.handoffContinueHint, icon: Scissors },
  ];

  return (
    <section className="creation-home-handoff" data-testid="creation-handoff" aria-label={c.handoffTitle}>
      <button type="button" className="creation-import" data-testid="create-import" onClick={props.onImport}>
        <Images className="size-4" aria-hidden />
        <span>
          <strong>{copy.composer.importClip}</strong>
          <small>{c.importLead}</small>
        </span>
      </button>
      <div className="creation-handoff-grid">
        {cards.map((card) => {
          const Icon = card.icon;
          return (
            <button
              key={card.id}
              type="button"
              className="creation-handoff-card"
              onClick={() => {
                if (card.id === "continue") {
                  if (cuts[0]?.id) props.onOpenCut(cuts[0].id);
                  else props.onImport();
                  return;
                }
                props.onHandoff(card.id);
              }}
            >
              <Icon className="size-3.5" aria-hidden />
              <span>
                <strong>{card.title}</strong>
                <small>{card.hint}</small>
              </span>
            </button>
          );
        })}
      </div>
      <div className="creation-recent">
        <h2>{c.recent}</h2>
        {cuts.length === 0 ? <p>{c.recentEmpty}</p> : (
          <ul>
            {cuts.map((row) => (
              <li key={row.id}>
                <button type="button" onClick={() => row.id && props.onOpenCut(row.id)}>
                  <span>{row.title || c.cutTitle}</span>
                  {row.duration ? <em>{row.duration.toFixed(1)}s</em> : null}
                </button>
              </li>
            ))}
          </ul>
        )}
      </div>
    </section>
  );
}
