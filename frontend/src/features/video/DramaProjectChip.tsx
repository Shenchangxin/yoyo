/**
 * 本文件拥有短剧系列和集的选择菜单。不拥有短剧看板。
 * 新建系列、新建集和绑定会话会改服务端。
 */
import { useCallback, useEffect, useState } from "react";
import { Check, ChevronDown, Film, LayoutGrid, Plus } from "lucide-react";
import { toast } from "sonner";
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuLabel, DropdownMenuSeparator, DropdownMenuTrigger } from "../../components/ui/dropdown-menu";
import { cn } from "../../lib/utils";
import { useCopy } from "../../lib/i18n";
import { useUI } from "../../lib/store";
import * as api from "../../lib/client";
import { defaultDramaStyle, useDramaSelection } from "./workshop-store";
import { bindVideoThread } from "./bind";
import { shownEpisodeTitle, shownSeriesTitle } from "./drama-lib";

type Drama = { id: string; title: string };
type Episode = { id: string; drama_id: string; title: string; episode_number?: number };

const ghost =
  "h-6 min-w-0 max-w-[9.5rem] gap-1 rounded-md border-transparent bg-transparent px-1.5 text-[11px] font-medium text-muted hover:bg-lift hover:text-foreground disabled:pointer-events-none disabled:opacity-30 [&_svg]:size-3 [&_svg]:opacity-70";

/** DramaProjectChip 列出系列和集，并能新建或打开看板。props 的 disabled 传给触发按钮。 */
export function DramaProjectChip(props: { disabled?: boolean }) {
  const copy = useCopy();
  const dramaId = useDramaSelection(/** s 取出 dramaId。 */ (s) => s.dramaId);
  const episodeId = useDramaSelection(/** s 取出 episodeId。 */ (s) => s.episodeId);
  const setDramaId = useDramaSelection(/** s 取出 setDramaId。 */ (s) => s.setDramaId);
  const setEpisodeId = useDramaSelection(/** s 取出 setEpisodeId。 */ (s) => s.setEpisodeId);
  const setVideoBoard = useUI(/** s 取出 setVideoBoard。 */ (s) => s.setVideoBoard);
  const sessionId = useUI(/** s 取出 videoThreadId。 */ (s) => s.videoThreadId);
  const [dramas, setDramas] = useState<Drama[]>([]);
  const [episodes, setEpisodes] = useState<Episode[]>([]);

  /** 读取系列列表。 */
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
    void api.video.episodes(dramaId).then(/** list 是数组才写入集列表。 */ (list) => {
      setEpisodes(Array.isArray(list) ? list : []);
    }).catch(() => setEpisodes([]));
  }, [dramaId]);

  const drama = dramas.find(/** d 的 id 和当前系列比对。 */ (d) => d.id === dramaId);
  const episode = episodes.find(/** e 的 id 和当前集比对。 */ (e) => e.id === episodeId);
  const label = episode
    ? shownEpisodeTitle(episode.title, episode.episode_number, copy.video.episodeN)
    : drama
      ? shownSeriesTitle(drama.title, copy.video.untitled)
      : copy.video.noProject;

  /** 按当前风格新建系列和一集，比例是 9:16。有会话就绑定。 */
  async function createDrama() {
    try {
      const d = await api.video.createDrama({ title: "", style: defaultDramaStyle(), aspect_ratio: "9:16" });
      setDramaId(d.id);
      const created = await api.video.createEpisode(d.id, "", "");
      setEpisodeId(created.id);
      if (sessionId) void bindVideoThread(sessionId);
      await load();
    } catch (e) {
      toast.error(api.errMessage(e));
    }
  }

  /** 在当前系列下新建一集。有会话就绑定。 */
  async function createEpisode() {
    if (!dramaId) return;
    try {
      const created = await api.video.createEpisode(dramaId, "", "");
      setEpisodeId(created.id);
      if (sessionId) void bindVideoThread(sessionId);
      const list = await api.video.episodes(dramaId);
      setEpisodes(Array.isArray(list) ? list : []);
    } catch (e) {
      toast.error(api.errMessage(e));
    }
  }

  /** 选中这一集。有会话就绑定。d 的 id 写成当前系列。ep 的 id 写成当前集。 */
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
        {dramas.map(/** d 交给 DramaGroup，id 和当前系列相同才带上当前集。 */ (d) => (
          <DramaGroup
            key={d.id}
            drama={d}
            activeEpisode={d.id === dramaId ? episodeId : ""}
            copyLabel={copy.video.untitledEp}
            episodeN={copy.video.episodeN}
            untitled={copy.video.untitled}
            onPick={pickEpisode}
          />
        ))}
      </DropdownMenuContent>
    </DropdownMenu>
  );
}

/** 展开一个系列下的集。props 的 drama 用来取集和标题，activeEpisode 和集 id 相同就打勾，copyLabel、episodeN、untitled 显示文案，onPick 带上系列和这一集。 */
function DramaGroup(props: {
  drama: Drama;
  activeEpisode: string;
  copyLabel: string;
  episodeN: string;
  untitled: string;
  onPick: (d: Drama, ep: Episode) => void;
}) {
  const [eps, setEps] = useState<Episode[]>([]);
  useEffect(() => {
    void api.video.episodes(props.drama.id).then(/** list 是数组才写入。 */ (list) => {
      setEps(Array.isArray(list) ? list : []);
    }).catch(() => setEps([]));
  }, [props.drama.id]);

  return (
    <>
      <DropdownMenuLabel className="truncate">{shownSeriesTitle(props.drama.title, props.untitled)}</DropdownMenuLabel>
      {eps.length ? eps.map(/** ep 显示标题，点选时和系列一起交给 onPick。 */ (ep) => (
        <DropdownMenuItem key={ep.id} onSelect={() => props.onPick(props.drama, ep)}>
          <span className="min-w-0 flex-1 truncate pl-1">{shownEpisodeTitle(ep.title, ep.episode_number, props.episodeN)}</span>
          {props.activeEpisode === ep.id ? <Check className="size-3.5 shrink-0" /> : null}
        </DropdownMenuItem>
      )) : (
        <div className="px-2.5 py-1 text-[11px] text-muted">{props.copyLabel}</div>
      )}
    </>
  );
}
