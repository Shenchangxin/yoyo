/**
 * 本文件拥有创作入口的模式切换按钮。不拥有各模式的桌子。
 * 它只改本地模式，不改服务端。
 */
import { AudioLines, Clapperboard, ListOrdered, MessagesSquare, Mic, Presentation, UserRound, Users } from "lucide-react";
import { useCopy } from "../../lib/i18n";
import { useCreateMode, type CreateMode } from "./create-mode";

const modes: { id: CreateMode; icon: typeof Clapperboard }[] = [
  { id: "clip", icon: Clapperboard },
  { id: "talking", icon: UserRound },
  { id: "drama", icon: Users },
  { id: "podcast", icon: MessagesSquare },
  { id: "interview", icon: Mic },
  { id: "ranking", icon: ListOrdered },
  { id: "explainer", icon: Presentation },
  { id: "narration", icon: AudioLines },
];

/** CreateModeTabs 在成片、榜单、播客、访谈之间切换。 */
export function CreateModeTabs() {
  const copy = useCopy();
  const c = copy.video.conductor;
  const film = copy.video.film;
  const mode = useCreateMode(/** s 取出 mode。 */ (s) => s.mode);
  const setMode = useCreateMode(/** s 取出 setMode。 */ (s) => s.setMode);
  const label: Record<CreateMode, string> = {
    clip: c.modeClip,
    talking: film.talking,
    drama: film.drama,
    podcast: film.podcast,
    interview: film.interview,
    ranking: film.ranking,
    explainer: film.explainer,
    narration: film.narration,
  };
  return (
    <div className="create-mode-tabs" role="group" aria-label={c.modeLabel} data-testid="create-mode-tabs">
      {modes.map(/** item 用 id 取文案和选中态，用 icon 画图标，点选时把 id 交给 setMode。 */ (item) => {
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
