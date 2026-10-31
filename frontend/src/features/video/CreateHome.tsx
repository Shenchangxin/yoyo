/**
 * 本文件拥有已经打开的成片和翻拍稿。新的创作从对话框里的技能开始。
 * 不拥有剪辑台和成片台的编辑。它只改本地指针，写入交给回调。
 */
import type { FilmMode } from "./film-cast";
import { FilmBoard } from "./FilmBoard";
import { useFilmDesk } from "./film-desk";
import { RemakeDraft } from "./RemakeDraft";
import { useRemakeDesk } from "./remake-desk";

/** CreateHome 在成片或翻拍稿打开时画出它们。props 的 onOpenCut 打开一条剪辑。 */
export function CreateHome(props: {
  onImport: () => void;
  onRemake: () => void;
  onFilm: (structure: FilmMode, description?: string) => void;
  onHandoff: (dest: "studio" | "drama" | "canvas") => void;
  onOpenCut: (id: string) => void;
  onPrompt?: (text: string) => void;
  draftKey?: string;
}) {
  const openFilm = useFilmDesk(/** s 取出 project。 */ (s) => s.project);
  const openDraft = useRemakeDesk(/** s 取出 draft。 */ (s) => s.draft);
  if (!openFilm?.id && !openDraft?.id) return null;
  return (
    <section className="creation-home-handoff" data-testid="creation-handoff">
      <FilmBoard onOpenCut={props.onOpenCut} />
      <RemakeDraft onOpenCut={props.onOpenCut} />
    </section>
  );
}
