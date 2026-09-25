import { useCallback, useEffect, useState } from "react";
import { Check, ChevronDown, Film, LayoutGrid, Plus } from "lucide-react";
import { toast } from "sonner";
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuLabel, DropdownMenuSeparator, DropdownMenuTrigger } from "../../components/ui/dropdown-menu";
import { cn } from "../../lib/utils";
import { useCopy } from "../../lib/i18n";
import { useUI } from "../../lib/store";
import * as api from "../../lib/client";
import { useDramaSelection } from "./workshop-store";
import { bindVideoThread } from "./bind";

type Drama = { id: string; title: string };
type Episode = { id: string; drama_id: string; title: string };

const ghost =
  "h-6 min-w-0 max-w-[9.5rem] gap-1 rounded-md border-transparent bg-transparent px-1.5 text-[11px] font-medium text-muted hover:bg-lift hover:text-foreground disabled:pointer-events-none disabled:opacity-30 [&_svg]:size-3 [&_svg]:opacity-70";

export function DramaProjectChip(props: { disabled?: boolean }) {
  const copy = useCopy();
  const dramaId = useDramaSelection((s) => s.dramaId);
  const episodeId = useDramaSelection((s) => s.episodeId);
  const setDramaId = useDramaSelection((s) => s.setDramaId);
  const setEpisodeId = useDramaSelection((s) => s.setEpisodeId);
  const setVideoBoard = useUI((s) => s.setVideoBoard);
  const sessionId = useUI((s) => s.videoThreadId);
  const [dramas, setDramas] = useState<Drama[]>([]);
  const [episodes, setEpisodes] = useState<Episode[]>([]);

  const load = useCallback(async () => {
    const list = (await api.video.listDramas()) as Drama[];
    setDramas(Array.isArray(list) ? list : []);
  }, []);

  useEffect(() => {
    void load().catch(() => {});
  }, [load]);

  useEffect(() => {
    if (!dramaId) {
      setEpisodes([]);
      return;
    }
    void api.video.episodes(dramaId).then((list) => {
      setEpisodes(Array.isArray(list) ? list : []);
    }).catch(() => setEpisodes([]));
  }, [dramaId]);

  const drama = dramas.find((d) => d.id === dramaId);
  const episode = episodes.find((e) => e.id === episodeId);
  const label = episode?.title || drama?.title || copy.video.noProject;

  async function createDrama() {
    try {
      const d = await api.video.createDrama({ title: copy.video.untitled, style: "3d", aspect_ratio: "16:9" });
      setDramaId(d.id);
      const created = await api.video.createEpisode(d.id, copy.video.untitledEp, "");
      setEpisodeId(created.id);
      if (sessionId) void bindVideoThread(sessionId);
      await load();
    } catch (e) {
      toast.error(api.errMessage(e));
    }
  }

  async function createEpisode() {
    if (!dramaId) return;
    try {
      const created = await api.video.createEpisode(dramaId, copy.video.untitledEp, "");
      setEpisodeId(created.id);
      if (sessionId) void bindVideoThread(sessionId);
      const list = await api.video.episodes(dramaId);
      setEpisodes(Array.isArray(list) ? list : []);
    } catch (e) {
      toast.error(api.errMessage(e));
    }
  }

  function pickEpisode(d: Drama, ep: Episode) {
    setDramaId(d.id);
    setEpisodeId(ep.id);
    if (sessionId) void bindVideoThread(sessionId);
  }

  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <button
          type="button"
          data-testid="drama-project-chip"
          className={cn(ghost, "inline-flex items-center", (drama || episode) && "text-foreground")}
          aria-label={copy.video.pickEpisode}
          disabled={props.disabled}
        >
          <Film className="size-3 shrink-0" aria-hidden />
          <span className="truncate">{label}</span>
          <ChevronDown className="size-3 shrink-0 opacity-70" />
        </button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="start" side="top" className="w-72 max-h-80 overflow-y-auto">
        <DropdownMenuItem data-testid="drama-open-board" onSelect={() => setVideoBoard(true)}>
          <LayoutGrid className="size-3.5 shrink-0 opacity-70" aria-hidden />
          {copy.video.openBoard}
        </DropdownMenuItem>
        <DropdownMenuItem onSelect={() => void createDrama()}>
          <Plus className="size-3.5 shrink-0 opacity-70" aria-hidden />
          {copy.video.newDrama}
        </DropdownMenuItem>
        {dramaId ? (
          <DropdownMenuItem onSelect={() => void createEpisode()}>
            <Film className="size-3.5 shrink-0 opacity-70" aria-hidden />
            {copy.video.newEpisode}
          </DropdownMenuItem>
        ) : null}
        {dramas.length ? <DropdownMenuSeparator /> : null}
        {dramas.map((d) => (
          <DramaGroup
            key={d.id}
            drama={d}
            activeEpisode={d.id === dramaId ? episodeId : ""}
            copyLabel={copy.video.untitledEp}
            onPick={pickEpisode}
          />
        ))}
      </DropdownMenuContent>
    </DropdownMenu>
  );
}

function DramaGroup(props: {
  drama: Drama;
  activeEpisode: string;
  copyLabel: string;
  onPick: (d: Drama, ep: Episode) => void;
}) {
  const [eps, setEps] = useState<Episode[]>([]);
  useEffect(() => {
    void api.video.episodes(props.drama.id).then((list) => {
      setEps(Array.isArray(list) ? list : []);
    }).catch(() => setEps([]));
  }, [props.drama.id]);

  return (
    <>
      <DropdownMenuLabel className="truncate">{props.drama.title || props.drama.id}</DropdownMenuLabel>
      {eps.length ? eps.map((ep) => (
        <DropdownMenuItem key={ep.id} onSelect={() => props.onPick(props.drama, ep)}>
          <span className="min-w-0 flex-1 truncate pl-1">{ep.title || props.copyLabel}</span>
          {props.activeEpisode === ep.id ? <Check className="size-3.5 shrink-0" /> : null}
        </DropdownMenuItem>
      )) : (
        <div className="px-2.5 py-1 text-[11px] text-muted">{props.copyLabel}</div>
      )}
    </>
  );
}
