/**
 * 本文件拥有创作入口的模式切换按钮。不拥有各模式的桌子。
 * 它只改本地模式，不改服务端。
 */
import { Clapperboard, ListOrdered, MessagesSquare, Mic } from "lucide-react";
import { useCopy } from "../../lib/i18n";
import { useCreateMode, type CreateMode } from "./create-mode";

const modes: { id: CreateMode; icon: typeof Clapperboard }[] = [
  { id: "clip", icon: Clapperboard },
  { id: "ranking", icon: ListOrdered },
  { id: "podcast", icon: MessagesSquare },
  { id: "interview", icon: Mic },
];

/** CreateModeTabs 在成片、榜单、播客、访谈之间切换。 */
export function CreateModeTabs() {
  const copy = useCopy();
  const c = copy.video.conductor;
  const film = copy.video.film;
  const mode = useCreateMode((s) => s.mode);
  const setMode = useCreateMode((s) => s.setMode);
  const label: Record<CreateMode, string> = {
    clip: c.modeClip,
    ranking: film.ranking,
    podcast: film.podcast,
    interview: film.interview,
  };
  return (
    <div className="create-mode-tabs" role="group" aria-label={c.modeLabel} data-testid="create-mode-tabs">
      {modes.map((item) => {
        const Icon = item.icon;
        const pressed = item.id === mode;
        return (
          <button
            key={item.id}
            type="button"
            data-testid={`create-mode-${item.id}`}
            aria-pressed={pressed}
            onClick={() => setMode(item.id)}
          >
            <Icon aria-hidden />
            <span>{label[item.id]}</span>
          </button>
        );
      })}
    </div>
  );
}
