/**
 * 本文件拥有创作入口上的技能。点一下把对应技能写进对话框。
 * 不拥有发送。发送仍是对话框自己的事。
 */
import { AudioLines, Clapperboard, Image, Film } from "lucide-react";
import { useCopy } from "../../lib/i18n";

/** CreateSkills 把剪视频、生图、生视频和配音写进当前草稿。onPrompt 接收整段要放进对话框的字。 */
export function CreateSkills(props: { onPrompt: (text: string) => void }) {
  const c = useCopy().video.conductor;
  const skills: { id: string; label: string; text: string; icon: typeof Film }[] = [
    { id: "cut-edit", label: c.skillCut, text: c.skillCutText, icon: Clapperboard },
    { id: "studio-storyboard", label: c.skillPicture, text: c.skillPictureText, icon: Image },
    { id: "hypit", label: c.skillFilm, text: c.skillFilmText, icon: Film },
    { id: "cut-dub", label: c.skillDub, text: c.skillDubText, icon: AudioLines },
  ];
  return (
    <div className="create-skill-row" role="group" aria-label={c.skillsLabel} data-testid="create-skill-row">
      {skills.map(/** item 用 id 做测试标记，用 text 交给 onPrompt。 */ (item) => {
        const Icon = item.icon;
        return (
          <button
            key={item.id}
            type="button"
            data-testid={`create-skill-${item.id}`}
            onClick={() => props.onPrompt(item.text)}
          >
            <Icon aria-hidden />
            <span>{item.label}</span>
          </button>
        );
      })}
    </div>
  );
}
