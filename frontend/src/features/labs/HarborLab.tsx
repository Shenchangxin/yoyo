// 本文件拥有评测报告的三步结果和再跑入口。不拥有评测怎么跑。调用方传入 report、best 和 onRun。

import { useEffect, useState } from "react";
import { ChevronDown } from "lucide-react";
import { asArray, num, pick, str } from "../../lib/normalize";
import { readHarborMetrics } from "../../lib/harness-refs";
import { checkHeadline } from "../../lib/harness-copy";
import { formatDurationMs, formatMoney, useCopy } from "../../lib/i18n";
import { cn } from "../../lib/utils";
import { Button } from "../../components/ui/button";
import { Input } from "../../components/ui/input";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "../../components/ui/dropdown-menu";
import { PresenceModuleLoading } from "../presence";
import type { HarborKind } from "../../lib/protocol";
import { LabCard, LabChip, LabFrame, LabTable, StoryRow } from "./LabFrame";
import { useUI } from "../../lib/store";

/** HarborLab 忙时只显示加载。安全项始终可跑；其余种类和模型输入只在实验室开关打开时出现。非实验室只列出未通过的结果。props 读取 report、busy、models、error、lastKind、best，调用 onRun，调用 onModels。 */
export function HarborLab(props: {
  busy: string | null;
  error?: string;
  lastKind?: HarborKind;
  report: any;
  best: any;
  models: string;
  onModels: (v: string) => void;
  onRun: (kind: HarborKind) => void;
}) {
  const copy = useCopy();
  const lab = useUI((s) => s.showHarnessLab);
  const metrics = readHarborMetrics(props.report);
  const results = asArray(pick(props.report, "results", "Results"));
  const inP = metrics?.heldInPass ?? 0;
  const inT = metrics?.heldInTotal ?? 0;
  const outP = metrics?.heldOutPass ?? 0;
  const outT = metrics?.heldOutTotal ?? 0;
  const safety = metrics?.safetyFail ?? 0;
  const why = safety > 0 ? copy.harbor.whySafety : props.report ? copy.harbor.whyEvidence : "";
  const headline = checkHeadline(copy, metrics);
  const [modelsOpen, setModelsOpen] = useState(false);
  // 忙着时打开全局模块加载，卸下或不再忙时关掉。
  useEffect(() => {
    if (!props.busy) return;
    useUI.getState().setModuleLoading(true);
    return () => useUI.getState().setModuleLoading(false);
  }, [props.busy]);

  if (props.busy) {
    return (
      <LabFrame>
        <PresenceModuleLoading label={props.busy} />
      </LabFrame>
    );
  }

  const visibleResults = lab ? results : results.filter((r: any) => !pick(r, "pass", "Pass"));
  const usd = num(pick(props.report, "usd", "USD"));
  const wall = num(pick(props.report, "wall_ms", "WallMs"));

  return (
    <LabFrame>
      <div className="mb-[var(--space-section)] flex max-w-sm flex-col items-stretch gap-[var(--space-group)]">
        <div className="flex items-center justify-end gap-2">
        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <Button variant="outline" disabled={!!props.busy}>
              {copy.rsi.moreEvals}
              <ChevronDown className="size-3.5" aria-hidden />
            </Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent align="end">
            <DropdownMenuItem onSelect={() => props.onRun("safety")}>{copy.harbor.runSafety}</DropdownMenuItem>
            {lab ? (
              <>
                <DropdownMenuItem onSelect={() => props.onRun("sealed")}>{copy.harbor.runSealed}</DropdownMenuItem>
                <DropdownMenuItem onSelect={() => props.onRun("transfer")}>{copy.harbor.runTransfer}</DropdownMenuItem>
                <DropdownMenuItem onSelect={() => props.onRun("index")}>{copy.harbor.runIndex}</DropdownMenuItem>
                <DropdownMenuItem onSelect={() => props.onRun("behavior")}>{copy.harbor.runBehavior}</DropdownMenuItem>
                <DropdownMenuItem onSelect={() => props.onRun("tb")}>{copy.harbor.runTb}</DropdownMenuItem>
                <DropdownMenuItem onSelect={() => props.onRun("bon")}>{copy.harbor.runBon}</DropdownMenuItem>
                <DropdownMenuItem onSelect={() => setModelsOpen(true)}>{copy.harbor.runModels}</DropdownMenuItem>
              </>
            ) : null}
          </DropdownMenuContent>
        </DropdownMenu>
        <Button disabled={!!props.busy} onClick={() => props.onRun("suite")}>{copy.harbor.runSuite}</Button>
        </div>
        {modelsOpen && lab ? (
          <label className="flex flex-col gap-[var(--space-item)] text-[13px] font-medium text-foreground">
            {copy.harbor.runModels}
            <Input className="h-8 w-full font-normal" placeholder={copy.harbor.modelsPh} value={props.models} onChange={(e) => props.onModels(e.target.value)} />
            <span className="mt-[var(--space-item)] flex justify-end">
              <Button variant="outline" disabled={!!props.busy} onClick={() => props.onRun("models")}>{copy.harbor.runModels}</Button>
            </span>
          </label>
        ) : null}
      </div>
      {props.error ? (
        <div className="mb-4 flex items-center gap-3 rounded-md bg-danger/8 px-4 py-3 text-[13px] text-danger">
          <span>{props.error}</span>
          <Button size="sm" variant="lift" onClick={() => props.onRun(props.lastKind || "suite")}>{copy.harbor.retry}</Button>
        </div>
      ) : null}
      {props.report ? (
        <>
          <p className={cn("mb-4 text-[15px] font-medium tracking-[-0.02em]", safety > 0 && "text-danger")}>
            {copy.harbor.verdict}. {safety > 0 ? copy.harbor.verdictBlock : inT && outT && inP === inT && outP === outT ? copy.harbor.verdictPromote : copy.harbor.whyEvidence}
          </p>
          <div className="mb-4 overflow-hidden rounded-md bg-sidebar/40">
            <StoryRow
              step="1"
              label={copy.harbor.heldIn}
              detail={copy.harbor.storyHeldIn}
              value={`${inP}/${inT}`}
              ok={inT === 0 || inP === inT}
            />
            <StoryRow
              step="2"
              label={copy.harbor.heldOut}
              detail={copy.harbor.storyHeldOut}
              value={`${outP}/${outT}`}
              ok={outT === 0 || outP === outT}
            />
            <StoryRow
              step="3"
              label={copy.harbor.safety}
              detail={copy.harbor.storySafety}
              value={String(safety)}
              ok={safety === 0}
              last
            />
          </div>
          <p className="mb-4 text-[13px] text-muted">
            {formatMoney(usd)} · {formatDurationMs(wall)}
            {lab ? (
              <span className="font-mono text-[12px]">
                {` · ${str(pick(props.report, "snapshot", "Snapshot")).slice(0, 16) || "—"}`}
                {str(pick(props.report, "model_fingerprint", "ModelFingerprint")) ? ` · ${str(pick(props.report, "model_fingerprint", "ModelFingerprint"))}` : ""}
                {` · ${copy.labs.tokens} ${num(pick(props.report, "tokens_in", "TokensIn"))}/${num(pick(props.report, "tokens_out", "TokensOut"))}`}
              </span>
            ) : null}
          </p>
          {why && safety > 0 ? <p className="mb-4 rounded-md bg-danger/8 px-4 py-3 text-[13px] text-danger">{why}</p> : null}
          {visibleResults.length === 0 ? (
            <p className="text-[13px] text-muted">{results.length ? headline : copy.harbor.noResults}</p>
          ) : (
            <LabTable>
              <thead className="text-[11px] text-muted">
                <tr>
                  <th className="px-4 py-2 font-medium">{copy.harbor.task}</th>
                  {lab ? <th className="font-medium">{copy.harbor.kind}</th> : null}
                  <th className="font-medium">{copy.harbor.pass}</th>
                  <th className="font-medium">{copy.harbor.error}</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-border/80">
                {visibleResults.map((r: any) => (
                  <tr key={str(pick(r, "id", "ID"))}>
                    <td className="px-4 py-2">{str(pick(r, "id", "ID"))}</td>
                    {lab ? <td className="text-muted">{str(pick(r, "kind", "Kind"), "—")}</td> : null}
                    <td><LabChip ok={!!pick(r, "pass", "Pass")}>{!!pick(r, "pass", "Pass") ? copy.harbor.pass : copy.harbor.fail}</LabChip></td>
                    <td className="text-muted">{str(pick(r, "error", "Error"))}</td>
                  </tr>
                ))}
              </tbody>
            </LabTable>
          )}
        </>
      ) : <p className="text-[13px] text-muted">{copy.harbor.empty}</p>}
      {lab && (props.best?.reports || props.best?.Reports) ? (
        <div className="mt-4 grid gap-3 sm:grid-cols-2">
          {asArray(pick(props.best, "reports", "Reports")).map((r: any, i: number) => {
            const m = pick(r, "metrics", "Metrics") || {};
            return (
              <LabCard key={i}>
                <div className="text-[13px] font-medium">{copy.harbor.runN} {i + 1}</div>
                <div className="mt-1 text-[11px] text-muted">
                  {copy.labs.in} {num(pick(m, "held_in_pass", "HeldInPass"))}/{num(pick(m, "held_in_total", "HeldInTotal"))} · {copy.labs.out} {num(pick(m, "held_out_pass", "HeldOutPass"))}/{num(pick(m, "held_out_total", "HeldOutTotal"))}
                </div>
              </LabCard>
            );
          })}
        </div>
      ) : null}
    </LabFrame>
  );
}
