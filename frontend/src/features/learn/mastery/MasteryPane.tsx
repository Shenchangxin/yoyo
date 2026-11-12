/**
 * 拥有：掌握这条泳道：目标列表、新建目标，以及打开一条路径后的看板。路径有变化时自己重读。
 * 不拥有：当前打开的是哪条路径，那由调用方持有，导师对话的范围跟着它走。也不拥有发送，和导师说话经 onSay。
 */

import { useCallback, useEffect, useState } from "react";
import { ListChecks } from "lucide-react";
import { useCopy } from "../../../lib/i18n";
import * as api from "../../../lib/client";
import * as learn from "../../../lib/learn-api";
import type { PathSummary, PathView } from "../../../lib/learn-api";
import { subscribeLearn } from "../../../lib/learn-events";
import { fill } from "../../../lib/learn-format";
import { errMessage } from "../../../lib/normalize";
import { Button } from "../../../components/ui/button";
import { EmptyState } from "../../../components/ui/empty-state";
import { ProgressHairline } from "../../../components/ui/progress-hairline";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "../../../components/ui/select";
import { LearnShell } from "../LearnShell";
import { PathBoard } from "./PathBoard";

type LibraryOption = { id: string; name: string };

const NO_LIBRARY = "none";

/**
 * MasteryPane 画出目标列表或一条路径的看板。active 是当前打开的路径，onActive 换掉它。
 * sessionId 是这条路径的导师对话。onSay 把一句话发给导师。
 */
export function MasteryPane(props: {
  active: PathView | null;
  onActive: (path: PathView | null) => void;
  sessionId?: string;
  onSay: (text: string) => void;
}) {
  const L = useCopy().learn.mastery;
  const { active, onActive } = props;
  const [list, setList] = useState<PathSummary[]>([]);
  const [loaded, setLoaded] = useState(false);
  const [err, setErr] = useState("");
  const [creating, setCreating] = useState(false);
  const [goal, setGoal] = useState("");
  const [libraryId, setLibraryId] = useState(NO_LIBRARY);
  const [libs, setLibs] = useState<LibraryOption[]>([]);
  const [busy, setBusy] = useState(false);
  const activeId = active?.id || "";

  const loadList = useCallback(async () => {
    try {
      setList(await learn.paths());
      setErr("");
    } catch (e) {
      setErr(errMessage(e));
    } finally {
      setLoaded(true);
    }
  }, []);

  /** 重读当前路径。路径已经被删掉时回到列表。 */
  const reloadActive = useCallback(async () => {
    if (!activeId) return;
    try {
      onActive(await learn.path(activeId));
    } catch {
      onActive(null);
    }
  }, [activeId, onActive]);

  useEffect(() => {
    void loadList();
  }, [loadList]);

  useEffect(() => subscribeLearn(/** 路径有变化就重读列表；变的是当前这条时也重读看板。 */ (change) => {
    if (change.scope !== "path") return;
    void loadList();
    if (change.id && change.id === activeId) void reloadActive();
  }), [loadList, reloadActive, activeId]);

  useEffect(() => {
    if (!creating) return;
    void api.learnCall("learn.libraries").then(/** 笔记库不当参考文库。 */ (rows) => {
      const all = Array.isArray(rows) ? rows : [];
      setLibs(all.filter((lib) => lib?.id && lib.id !== "notes").map((lib) => ({ id: String(lib.id), name: String(lib.name || lib.id) })));
    }).catch(() => setLibs([]));
  }, [creating]);

  /** 打开一条路径。 */
  const open = async (id: string) => {
    setErr("");
    try {
      onActive(await learn.path(id));
    } catch (e) {
      setErr(errMessage(e));
    }
  };

  /** 按写下的目标建一条路径并打开。 */
  const create = async () => {
    const text = goal.trim();
    if (!text || busy) return;
    setBusy(true);
    setErr("");
    try {
      const made = await learn.createPath(text, libraryId === NO_LIBRARY ? "" : libraryId);
      setGoal("");
      setCreating(false);
      onActive(made);
      void loadList();
    } catch (e) {
      setErr(errMessage(e));
    } finally {
      setBusy(false);
    }
  };

  if (active) {
    return (
      <PathBoard
        path={active}
        sessionId={props.sessionId}
        onReload={async () => {
          await reloadActive();
          void loadList();
        }}
        onSay={props.onSay}
        onClose={() => onActive(null)}
        onDeleted={() => {
          onActive(null);
          void loadList();
        }}
      />
    );
  }

  return (
    <LearnShell
      title={L.title}
      subtitle={L.subtitle}
      action={<Button size="sm" className="whitespace-nowrap" aria-expanded={creating} onClick={() => setCreating((v) => !v)}>{L.newGoal}</Button>}
    >
      <div className="mx-auto flex max-w-[680px] flex-col px-8 py-6" data-testid="learn-mastery">
        {err ? <p className="mb-3 text-ui-footnote text-danger" aria-live="polite">{err}</p> : null}
        {creating ? (
          <form
            className="mb-6 flex flex-col gap-3"
            onSubmit={(e) => {
              e.preventDefault();
              void create();
            }}
          >
            <label className="flex flex-col gap-1">
              <span className="text-ui-footnote text-muted">{L.goalLabel}</span>
              <textarea
                className="min-h-20 w-full resize-y rounded-lg border border-border/60 bg-transparent px-3 py-2 text-ui-body outline-none focus-visible:border-accent/60"
                value={goal}
                placeholder={L.goalPh}
                autoFocus
                onChange={(e) => setGoal(e.target.value)}
              />
            </label>
            {libs.length > 0 ? (
              <label className="flex flex-col gap-1">
                <span className="text-ui-footnote text-muted">{L.libraryLabel}</span>
                <Select value={libraryId} onValueChange={setLibraryId}>
                  <SelectTrigger className="w-full max-w-xs" aria-label={L.libraryLabel}>
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value={NO_LIBRARY}>{L.libraryNone}</SelectItem>
                    {libs.map((lib) => <SelectItem key={lib.id} value={lib.id}>{lib.name}</SelectItem>)}
                  </SelectContent>
                </Select>
              </label>
            ) : null}
            <div className="flex items-center gap-2">
              <Button size="sm" type="submit" className="whitespace-nowrap" disabled={!goal.trim() || busy}>{L.create}</Button>
              <Button size="sm" variant="ghost" className="whitespace-nowrap" onClick={() => setCreating(false)}>{L.cancel}</Button>
            </div>
          </form>
        ) : null}
        {loaded && list.length === 0 && !creating ? <EmptyState title={L.empty} body={L.emptyHint} /> : null}
        <ul className="flex flex-col">
          {list.map((p) => {
            const total = p.counts.total;
            const badge = p.pending_question ? L.pendingBadge : p.due_reviews > 0 ? fill(L.dueBadge, { n: p.due_reviews }) : p.complete ? L.doneBadge : "";
            return (
              <li key={p.id}>
                <button type="button" onClick={() => { void open(p.id); }} className="flex w-full items-center gap-3 border-b border-border/60 py-3 text-left hover:bg-lift/40">
                  <span className="grid size-9 shrink-0 place-items-center rounded-lg bg-lift text-muted"><ListChecks className="size-4" aria-hidden /></span>
                  <span className="min-w-0 flex-1">
                    <span className="block truncate text-ui-body-l">{p.name}</span>
                    <span className="mt-0.5 flex items-center gap-2 text-ui-footnote text-muted">
                      <span className="tabular-nums">{p.counts.mastered}/{total}</span>
                      {badge ? <span className={p.complete && !p.pending_question && p.due_reviews === 0 ? "" : "text-warning"}>{badge}</span> : null}
                    </span>
                    {total > 0 ? <ProgressHairline className="mt-2" value={(p.counts.mastered / total) * 100} /> : null}
                  </span>
                </button>
              </li>
            );
          })}
        </ul>
      </div>
    </LearnShell>
  );
}
