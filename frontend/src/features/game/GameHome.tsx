import { useCallback, useEffect, useState, type ReactNode } from "react";
import { toast } from "sonner";
import { Plus, Trash2 } from "lucide-react";
import { Button } from "../../components/ui/button";
import { PresenceAnchor } from "../presence";
import { ConfirmDialog } from "../ConfirmDialog";
import { useCopy } from "../../lib/i18n";
import * as api from "../../lib/client";
import type { VideoProject } from "../../lib/protocol";
import { createGameWorld } from "./game-actions";
import { gameStageLabel, type GameRow } from "./game-model";

function asRows(v: unknown): GameRow[] {
  return Array.isArray(v) ? v.filter((r: GameRow) => r?.id) : [];
}

export function GameHome(props: {
  composer: ReactNode;
  onPrompt?: (text: string) => void;
  onOpenWorld: (p: VideoProject) => void;
}) {
  const copy = useCopy();
  const g = copy.video.gameDesk;
  const starters = copy.transcript.gameStarters;

  return (
    <div className="relative flex h-full min-h-0 flex-col overflow-hidden" data-testid="game-desk-home" data-skin-pane>
      <div className="home-stage empty-rise flex min-h-0 flex-1 flex-col overflow-auto" data-testid="empty-turn">
        <div className="mx-auto flex w-full max-w-[min(100%,var(--thread-measure))] shrink-0 flex-col px-5 pb-3 pt-6 sm:px-8">
          <PresenceAnchor id="home" size={96} className="mx-auto cursor-pointer" />
          <h1 className="mt-5 text-center text-[21px] font-semibold leading-[1.2] tracking-[-0.036em] text-pretty text-foreground">
            {copy.transcript.gameReady}
          </h1>
          <p className="mx-auto mt-2 max-w-[40ch] text-center text-[13px] leading-[1.55] text-muted">
            {g.homeLead}
          </p>
          <div className="mt-8 w-full">{props.composer}</div>
          <div className="mt-4 flex flex-wrap justify-center gap-1.5">
            {starters.map((s, i) => (
              <button
                type="button"
                key={s.label}
                className="starter-tile"
                style={{ animationDelay: `${80 + i * 50}ms` }}
                onClick={() => props.onPrompt?.(s.text)}
              >
                <span className="text-[13px] font-medium tracking-[-0.02em] text-foreground">{s.label}</span>
              </button>
            ))}
          </div>
        </div>
        <GameWorldList onOpenWorld={props.onOpenWorld} />
      </div>
    </div>
  );
}

function GameWorldList(props: {
  onOpenWorld: (p: VideoProject) => void;
}) {
  const copy = useCopy();
  const g = copy.video.gameDesk;
  const [rows, setRows] = useState<GameRow[]>([]);
  const [busy, setBusy] = useState("");
  const [kill, setKill] = useState<GameRow | null>(null);

  const load = useCallback(async () => {
    setRows(asRows(await api.video.listGames()));
  }, []);

  useEffect(() => {
    void load().catch(() => undefined);
  }, [load]);

  async function create() {
    if (busy) return;
    setBusy("create");
    try {
      const created = await createGameWorld({ title: copy.video.newGame });
      await load();
      props.onOpenWorld({ kind: "game", id: created.id, title: created.title || copy.video.newGame });
    } catch (e) {
      toast.error(api.errMessage(e));
    } finally {
      setBusy("");
    }
  }

  return (
    <div
      className="mx-auto w-full max-w-[min(100%,var(--thread-measure))] px-5 pb-8 sm:px-8"
      data-testid="game-library"
    >
      <div className="mt-8">
        <div className="flex items-center gap-2">
          <p className="min-w-0 flex-1 text-[11px] font-medium text-muted">{g.worlds}</p>
          <Button
            size="sm"
            variant="default"
            className="h-7 shrink-0 gap-1 text-[12px]"
            disabled={!!busy}
            data-testid="game-create"
            onClick={() => void create()}
          >
            <Plus className="size-3.5" aria-hidden />
            {busy === "create" ? g.creating : copy.video.newGame}
          </Button>
        </div>
        {rows.length ? (
          <ul
            className="mt-2 divide-y divide-border overflow-hidden rounded-xl border border-border bg-card"
            data-testid="game-recents"
          >
            {rows.map((row) => (
              <li key={row.id} className="flex items-stretch">
                <button
                  type="button"
                  className="flex min-w-0 flex-1 items-center gap-3 px-3 py-2.5 text-left hover:bg-lift/70"
                  data-testid={`game-recent-${row.id}`}
                  onClick={() => props.onOpenWorld({ kind: "game", id: row.id, title: row.title || copy.video.newGame })}
                >
                  <span className="min-w-0 flex-1">
                    <span className="block truncate text-[13px] font-medium">{row.title || copy.video.newGame}</span>
                    <span className="mt-0.5 block truncate text-[11px] text-muted">{row.intent || g.homeHint}</span>
                  </span>
                  <span className="shrink-0 text-[11px] text-muted">{gameStageLabel(row.stage, g)}</span>
                </button>
                <button
                  type="button"
                  className="shrink-0 px-2.5 text-muted hover:text-danger"
                  aria-label={g.delete}
                  data-testid={`game-recent-delete-${row.id}`}
                  onClick={() => setKill(row)}
                >
                  <Trash2 className="size-3.5" aria-hidden />
                </button>
              </li>
            ))}
          </ul>
        ) : (
          <p className="mt-2 text-[13px] text-muted">{copy.video.gameEmpty}</p>
        )}
      </div>
      <ConfirmDialog
        open={!!kill}
        title={g.delete}
        body={g.deleteBody}
        danger
        onCancel={() => setKill(null)}
        onConfirm={async () => {
          if (!kill) return;
          setBusy("delete");
          try {
            await api.video.deleteGame(kill.id);
            await load();
          } catch (e) {
            toast.error(api.errMessage(e));
          } finally {
            setBusy("");
            setKill(null);
          }
        }}
      />
    </div>
  );
}
