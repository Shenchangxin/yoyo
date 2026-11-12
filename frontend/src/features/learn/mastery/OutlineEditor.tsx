/**
 * 拥有：大纲的编辑表单：改章名和目标、加减知识点、选类型。
 * 不拥有：保存。改好的大纲交给调用方，由引擎决定哪些知识点保住进度。前置关系原样带着，这里不编辑。
 */

import { useState } from "react";
import { Plus, X } from "lucide-react";
import { useCopy } from "../../../lib/i18n";
import type { OutlineModule, OutlinePoint, PointKind } from "../../../lib/learn-api";
import { Button } from "../../../components/ui/button";
import { Input } from "../../../components/ui/input";

const KINDS: PointKind[] = ["memory", "procedure", "concept", "design"];

/** kindLabel 返回类型在当前语言里的名字。 */
export function kindLabel(L: ReturnType<typeof useCopy>["learn"]["mastery"], kind: string): string {
  if (kind === "memory") return L.kindMemory;
  if (kind === "procedure") return L.kindProcedure;
  if (kind === "design") return L.kindDesign;
  return L.kindConcept;
}

/**
 * OutlineEditor 编辑一份大纲。initial 是打开时的内容。hint 非空时显示在顶上，草稿用它说明还没写入。
 * busy 为真时保存按钮不可点。onSave 收到去掉空行后的大纲。onCancel 放弃修改。
 */
export function OutlineEditor(props: {
  initial: OutlineModule[];
  hint?: string;
  busy?: boolean;
  onSave: (modules: OutlineModule[]) => void;
  onCancel: () => void;
}) {
  const L = useCopy().learn.mastery;
  const [modules, setModules] = useState<OutlineModule[]>(() => (props.initial.length ? props.initial : [{ name: "", objective: "", points: [] }]));

  /** 改第 mi 章。 */
  const patchModule = (mi: number, patch: Partial<OutlineModule>) => {
    setModules((list) => list.map((m, i) => (i === mi ? { ...m, ...patch } : m)));
  };
  /** 改第 mi 章的第 pi 个知识点。 */
  const patchPoint = (mi: number, pi: number, patch: Partial<OutlinePoint>) => {
    setModules((list) => list.map((m, i) => (i === mi ? { ...m, points: m.points.map((p, j) => (j === pi ? { ...p, ...patch } : p)) } : m)));
  };
  /** 去掉没名字的知识点和既没名字也没知识点的章。 */
  const cleaned = () => modules
    .map((m) => ({ ...m, name: m.name.trim(), objective: (m.objective || "").trim(), points: m.points.filter((p) => p.name.trim()).map((p) => ({ ...p, name: p.name.trim() })) }))
    .filter((m) => m.name || m.points.length);
  const total = modules.reduce((n, m) => n + m.points.filter((p) => p.name.trim()).length, 0);

  return (
    <div className="flex flex-col gap-5" data-testid="learn-outline-editor">
      {props.hint ? <p className="text-ui-footnote text-muted">{props.hint}</p> : null}
      {modules.map((mod, mi) => (
        <section key={mod.id || `new-${mi}`} className="flex flex-col gap-2 border-b border-border/50 pb-4">
          <div className="flex items-center gap-2">
            <Input className="min-w-0 flex-1" value={mod.name} placeholder={L.moduleNamePh} aria-label={L.moduleNamePh} onChange={(e) => patchModule(mi, { name: e.target.value })} />
            <Button size="icon" variant="ghost" aria-label={L.removeModule} title={L.removeModule} onClick={() => setModules((list) => list.filter((_, i) => i !== mi))}>
              <X />
            </Button>
          </div>
          <Input value={mod.objective || ""} placeholder={L.objectivePh} aria-label={L.objectivePh} onChange={(e) => patchModule(mi, { objective: e.target.value })} />
          <ul className="flex flex-col gap-1.5 pl-3">
            {mod.points.map((pt, pi) => (
              <li key={pt.id || `new-${mi}-${pi}`} className="flex items-center gap-2">
                <Input className="min-w-0 flex-1" value={pt.name} placeholder={L.pointNamePh} aria-label={L.pointNamePh} onChange={(e) => patchPoint(mi, pi, { name: e.target.value })} />
                <select
                  className="h-8 shrink-0 rounded-lg border border-border/60 bg-transparent px-2 text-ui-footnote"
                  value={pt.kind}
                  aria-label={L.kindLabel}
                  onChange={(e) => patchPoint(mi, pi, { kind: e.target.value as PointKind })}
                >
                  {KINDS.map((k) => <option key={k} value={k}>{kindLabel(L, k)}</option>)}
                </select>
                <Button size="icon" variant="ghost" aria-label={L.removePoint} title={L.removePoint} onClick={() => patchModule(mi, { points: mod.points.filter((_, j) => j !== pi) })}>
                  <X />
                </Button>
              </li>
            ))}
          </ul>
          <div className="pl-3">
            <Button size="sm" variant="ghost" className="whitespace-nowrap" onClick={() => patchModule(mi, { points: [...mod.points, { name: "", kind: "concept" }] })}>
              <Plus />{L.addPoint}
            </Button>
          </div>
        </section>
      ))}
      <div>
        <Button size="sm" variant="ghost" className="whitespace-nowrap" onClick={() => setModules((list) => [...list, { name: "", objective: "", points: [] }])}>
          <Plus />{L.addModule}
        </Button>
      </div>
      <p className="text-ui-footnote text-muted">{L.removeWarn}</p>
      <div className="flex items-center gap-2">
        <Button size="sm" className="whitespace-nowrap" disabled={props.busy || total === 0} onClick={() => props.onSave(cleaned())}>{L.saveOutline}</Button>
        <Button size="sm" variant="ghost" className="whitespace-nowrap" disabled={props.busy} onClick={props.onCancel}>{L.discard}</Button>
      </div>
    </div>
  );
}
