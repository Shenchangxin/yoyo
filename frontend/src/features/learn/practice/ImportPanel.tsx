/**
 * 拥有：批量导入题目的两步：选文件或粘贴内容先预览，看过每一行的结果再提交。
 * 不拥有：解析和校验，那在引擎里。预览不写题库，提交才写。
 */

import { useState } from "react";
import { useCopy } from "../../../lib/i18n";
import * as api from "../../../lib/client";
import * as practice from "../../../lib/learn-practice";
import type { ImportPreview } from "../../../lib/learn-practice";
import { fill } from "../../../lib/learn-format";
import { splitTags } from "../../../lib/learn-practice-util";
import { errMessage } from "../../../lib/normalize";
import { Button } from "../../../components/ui/button";
import { Input } from "../../../components/ui/input";
import { Checkbox } from "../../../components/ui/checkbox";
import { cn } from "../../../lib/utils";
import { Field, textareaClass, typeLabel } from "./parts";

const SHOWN_ROWS = 60;

/** ImportPanel 画出导入的两步。onDone 在导入写入之后调用，带上一句结果。onCancel 收起面板。 */
export function ImportPanel(props: { onDone: (message: string) => void; onCancel: () => void }) {
  const L = useCopy().learn.drill;
  const [text, setText] = useState("");
  const [preview, setPreview] = useState<ImportPreview | null>(null);
  const [tags, setTags] = useState("");
  const [asMistakes, setAsMistakes] = useState(false);
  const [busy, setBusy] = useState(false);
  const [err, setErr] = useState("");

  /** 预览一份文件或一段粘贴的内容。 */
  const run = async (source: { path?: string; text?: string }) => {
    if (busy) return;
    setBusy(true);
    setErr("");
    try {
      setPreview(await practice.previewImport(source));
    } catch (e) {
      setErr(errMessage(e));
    } finally {
      setBusy(false);
    }
  };

  /** 让学习者选一份文件，选中后直接预览。 */
  const pick = async () => {
    const files = await api.pickFiles();
    if (files.length > 0) await run({ path: files[0] });
  };

  /** 把预览里合格的行写进题库。 */
  const commit = async () => {
    if (!preview || busy) return;
    setBusy(true);
    setErr("");
    try {
      const res = await practice.commitImport(preview.token, { tags: splitTags(tags), asMistakes });
      props.onDone(res.skipped > 0 ? fill(L.importSkipped, { n: res.imported, skipped: res.skipped }) : fill(L.imported, { n: res.imported }));
    } catch (e) {
      setErr(errMessage(e));
    } finally {
      setBusy(false);
    }
  };

  return (
    <section className="flex flex-col gap-4" data-testid="learn-import">
      <div>
        <p className="text-ui-body font-medium">{L.importTitle}</p>
        <p className="mt-1 text-ui-footnote leading-relaxed text-muted">{L.importHint}</p>
      </div>
      {err ? <p className="text-ui-footnote text-danger" aria-live="polite">{err}</p> : null}
      {!preview ? (
        <>
          <textarea className={cn(textareaClass, "min-h-28 font-mono text-ui-footnote")} value={text} placeholder={L.pastePh} aria-label={L.pastePh} onChange={(e) => setText(e.target.value)} />
          <div className="flex items-center gap-2">
            <Button size="sm" className="whitespace-nowrap" disabled={!text.trim() || busy} onClick={() => { void run({ text }); }}>{L.preview}</Button>
            <Button size="sm" variant="outline" className="whitespace-nowrap" disabled={busy} onClick={() => { void pick(); }}>{L.pickFile}</Button>
            <Button size="sm" variant="ghost" className="whitespace-nowrap" onClick={props.onCancel}>{L.cancel}</Button>
          </div>
        </>
      ) : (
        <>
          <p className="text-ui-body tabular-nums">{fill(L.previewStats, { valid: preview.valid, invalid: preview.invalid, dup: preview.duplicates })}</p>
          <ul className="max-h-72 overflow-auto border-y border-border/50">
            {preview.rows.slice(0, SHOWN_ROWS).map((row) => (
              <li key={row.line} className="flex items-baseline gap-3 border-b border-border/40 py-1.5 text-ui-footnote last:border-b-0">
                <span className="w-14 shrink-0 tabular-nums text-muted">{fill(L.rowLine, { n: row.line })}</span>
                <span className="min-w-0 flex-1 truncate">{row.question ? row.question.stem : row.error}</span>
                <span className={cn("shrink-0 whitespace-nowrap", row.error ? "text-danger" : row.duplicate ? "text-muted" : "text-success")}>
                  {row.error ? "" : row.duplicate ? L.rowDuplicate : `${typeLabel(L, row.question?.qtype || "")} · ${L.rowOk}`}
                </span>
              </li>
            ))}
          </ul>
          {preview.rows.length > SHOWN_ROWS ? <p className="text-ui-footnote text-muted">{fill(L.moreRows, { n: preview.rows.length - SHOWN_ROWS })}</p> : null}
          <Field label={L.tagsLabel}><Input value={tags} placeholder={L.tagsPh} onChange={(e) => setTags(e.target.value)} /></Field>
          <label className="flex items-center gap-2 text-ui-body">
            <Checkbox checked={asMistakes} onChange={(e) => setAsMistakes(e.target.checked)} />
            <span>{L.asMistakes}</span>
          </label>
          <div className="flex items-center gap-2">
            <Button size="sm" className="whitespace-nowrap" disabled={preview.valid === 0 || busy} onClick={() => { void commit(); }}>{fill(L.importN, { n: preview.valid })}</Button>
            <Button size="sm" variant="ghost" className="whitespace-nowrap" disabled={busy} onClick={() => setPreview(null)}>{L.startOver}</Button>
            <Button size="sm" variant="ghost" className="whitespace-nowrap" onClick={props.onCancel}>{L.cancel}</Button>
          </div>
        </>
      )}
    </section>
  );
}
