/**
 * 本文件拥有创作入口上的动作和技能。选视频和翻拍直接动手；剪视频、生图、生视频、配音把对应技能写进对话框。
 * 不拥有发送。发送仍是对话框自己的事。生视频按片型写一份简报，操作者改好主题再发。
 */
import { AudioLines, ChevronDown, Clapperboard, Film, FolderOpen, Image, Repeat2 } from "lucide-react";
import { useCopy } from "../../lib/i18n";
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuSeparator, DropdownMenuTrigger } from "../../components/ui/dropdown-menu";
import { filmCommission, readCast, type FilmMode } from "./film-cast";

const FILM_MODES: FilmMode[] = ["talking", "drama", "podcast", "interview", "ranking", "explainer", "narration"];

/** CreateSkills 画出选视频、翻拍两个动作，以及剪视频、生图、生视频、配音四个技能。props 的 onPrompt 接收整段要放进对话框的字，onImport 打开选视频，onRemake 打开翻拍。 */
export function CreateSkills(props: { onPrompt: (text: string) => void; onImport?: () => void; onRemake?: () => void }) {
  const copy = useCopy();
  const c = copy.video.conductor;
  const film = copy.video.film;
  const hints: Record<FilmMode, string> = {
    talking: film.talkingHint,
    drama: film.dramaHint,
    podcast: film.podcastHint,
    interview: film.interviewHint,
    ranking: film.rankingHint,
    explainer: film.explainerHint,
    narration: film.narrationHint,
  };
  const names: Record<FilmMode, string> = {
    talking: film.talking,
    drama: film.drama,
    podcast: film.podcast,
    interview: film.interview,
    ranking: film.ranking,
    explainer: film.explainer,
    narration: film.narration,
  };
  return (
    <div className="create-skill-row" role="group" aria-label={c.skillsLabel} data-testid="create-skill-row">
      {props.onImport ? (
        <button type="button" data-testid="create-import" title={c.pickVideoHint} onClick={props.onImport}>
          <FolderOpen aria-hidden />
          <span>{c.pickVideo}</span>
        </button>
      ) : null}
      {props.onRemake ? (
        <button type="button" data-testid="create-remake" title={c.remakeHint} onClick={props.onRemake}>
          <Repeat2 aria-hidden />
          <span>{c.remake}</span>
        </button>
      ) : null}
      <button type="button" data-testid="create-skill-cut-edit" title={c.skillPromptHint} onClick={() => props.onPrompt(c.skillCutText)}>
        <Clapperboard aria-hidden />
        <span>{c.skillCut}</span>
      </button>
      <button type="button" data-testid="create-skill-studio-storyboard" title={c.skillPromptHint} onClick={() => props.onPrompt(c.skillPictureText)}>
        <Image aria-hidden />
        <span>{c.skillPicture}</span>
      </button>
      <DropdownMenu>
        <DropdownMenuTrigger asChild>
          <button type="button" data-testid="create-skill-hypit" title={c.skillPromptHint}>
            <Film aria-hidden />
            <span>{c.skillFilm}</span>
            <ChevronDown aria-hidden />
          </button>
        </DropdownMenuTrigger>
        <DropdownMenuContent align="center" className="w-60" data-testid="create-film-modes">
          <DropdownMenuItem onSelect={() => props.onPrompt(c.skillFilmText)}>{c.skillFilm}</DropdownMenuItem>
          <DropdownMenuSeparator />
          {FILM_MODES.map(/** mode 选出片型名字和提示，点选时把这一种片子的简报写进对话框。 */ (mode) => (
            <DropdownMenuItem
              key={mode}
              data-testid={`create-film-${mode}`}
              title={hints[mode]}
              onSelect={() => props.onPrompt(filmCommission(mode, readCast(mode, film), hints[mode], {
                intro: film.briefIntro, format: film.briefFormat, relation: film.briefRelation, subject: film.briefSubject,
                left: film.briefLeft, ask: film.briefAsk, right: film.briefRight, reply: film.briefReply,
                speaker: film.briefSpeaker, outro: film.briefOutro, modeName: names[mode],
              }))}
            >
              {names[mode]}
            </DropdownMenuItem>
          ))}
        </DropdownMenuContent>
      </DropdownMenu>
      <button type="button" data-testid="create-skill-cut-dub" title={c.skillPromptHint} onClick={() => props.onPrompt(c.skillDubText)}>
        <AudioLines aria-hidden />
        <span>{c.skillDub}</span>
      </button>
    </div>
  );
}
