/**
 * 本文件拥有短片副驾标题条上的名称和下一步文案。不拥有聊天记录。
 * 它只读本地选择，不改服务端。短片编号只放在悬停提示里，不摊在标题下面。
 */
import { Sparkles } from "lucide-react";
import { useCopy } from "../../lib/i18n";
import { studioNextCopy, type StudioNextKey } from "./studio-next";
import { useStudioSelection } from "./workshop-store";

const NEXT_KEYS: StudioNextKey[] = ["collect", "produce", "setup", "script", "audio", "visuals", "compose", "pack", "open", "watch"];

/** StudioCopilotHeader 显示当前短片标题；没有标题时下一步落到收集。一键成片在跑时下一步是去看进度。 */
export function StudioCopilotHeader() {
  const copy = useCopy();
  const s = copy.video.studio;
  const title = useStudioSelection(/** st 取出 title。 */ (st) => st.title);
  const studioId = useStudioSelection(/** st 取出 studioId。 */ (st) => st.studioId);
  const nextKey = useStudioSelection(/** st 取出 nextKey。 */ (st) => st.nextKey);
  const key = NEXT_KEYS.includes(nextKey as StudioNextKey) ? nextKey as StudioNextKey : (title ? "script" : "collect");
  const nextLabel = studioNextCopy(s, key);

  return (
    <header className="flex shrink-0 items-start gap-2 border-b border-border/70 px-3 py-2" data-testid="studio-copilot-header">
      <Sparkles className="mt-0.5 size-3.5 shrink-0 opacity-70" aria-hidden />
      <div className="min-w-0 flex-1">
        <p className="text-ui-caption font-medium text-muted" data-testid="studio-copilot-label">{copy.video.studioCopilot}</p>
        <p className="truncate text-ui-body font-medium" data-testid="studio-copilot-bound" title={studioId ? `${s.inspectHint}\n${studioId}` : undefined}>
          {title || s.desk.unbound}
        </p>
        <p className="truncate text-ui-footnote text-muted" data-testid="studio-copilot-next">
          {title ? `${s.desk.boundHint} · ${nextLabel}` : (s.desk.unboundHint)}
        </p>
      </div>
    </header>
  );
}
