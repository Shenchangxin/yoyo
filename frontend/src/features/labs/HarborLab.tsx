import { useState } from "react";
import { ChevronDown } from "lucide-react";
import { asArray, num, pick, str } from "../../lib/normalize";
import { readHarborMetrics } from "../../lib/harness-refs";
import { useCopy } from "../../lib/i18n";
import { Button } from "../../components/ui/button";
import { Input } from "../../components/ui/input";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "../../components/ui/dropdown-menu";
import { LabCard, LabChip, LabFrame, LabStat, LabTable } from "./LabFrame";
import type { HarborKind } from "../../lib/protocol";

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
  const metrics = readHarborMetrics(props.report);
  const results = asArray(pick(props.report, "results", "Results"));
  const inP = metrics?.heldInPass ?? 0;
  const inT = metrics?.heldInTotal ?? 0;
  const outP = metrics?.heldOutPass ?? 0;
  const outT = metrics?.heldOutTotal ?? 0;
  const safety = metrics?.safetyFail ?? 0;
  const why = safety > 0 ? copy.harbor.whySafety : props.report ? copy.harbor.whyEvidence : "";
  const [modelsOpen, setModelsOpen] = useState(false);

  return (
    <LabFrame>
      <div className="mb-5 flex flex-wrap items-center gap-2 rounded-xl border border-border/80 bg-card p-2">
        <Button disabled={!!props.busy} onClick={() => props.onRun("suite")}>{copy.harbor.runSuite}</Button>
        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <Button variant="lift" disabled={!!props.busy}>
              {copy.rsi.moreEvals}
              <ChevronDown className="size-3.5" aria-hidden />
            </Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent align="start">
            <DropdownMenuItem onSelect={() => props.onRun("safety")}>{copy.harbor.runSafety}</DropdownMenuItem>
            <DropdownMenuItem onSelect={() => props.onRun("sealed")}>{copy.harbor.runSealed}</DropdownMenuItem>
            <DropdownMenuItem onSelect={() => props.onRun("transfer")}>{copy.harbor.runTransfer}</DropdownMenuItem>
            <DropdownMenuItem onSelect={() => props.onRun("tb")}>{copy.harbor.runTb}</DropdownMenuItem>
            <DropdownMenuItem onSelect={() => props.onRun("bon")}>{copy.harbor.runBon}</DropdownMenuItem>
            <DropdownMenuItem onSelect={() => setModelsOpen(true)}>{copy.harbor.runModels}</DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>
        {modelsOpen ? (
          <>
            <Input className="h-8 max-w-xs" placeholder={copy.harbor.modelsPh} value={props.models} onChange={(e) => props.onModels(e.target.value)} />
            <Button variant="lift" disabled={!!props.busy} onClick={() => props.onRun("models")}>{copy.harbor.runModels}</Button>
          </>
        ) : null}
      </div>
      {props.busy ? <p className="mb-4 text-[13px] text-accent">{props.busy}</p> : null}
      {!props.busy && props.error ? (
        <div className="mb-4 flex items-center gap-3 rounded-xl border border-danger/30 bg-danger/10 px-4 py-3 text-[13px] text-danger">
          <span>{props.error}</span>
          <Button size="sm" variant="lift" onClick={() => props.onRun(props.lastKind || "suite")}>{copy.harbor.retry}</Button>
        </div>
      ) : null}
      {props.report ? (
        <>
          <div className="mb-4 grid grid-cols-3 gap-3">
            <LabStat label={copy.harbor.heldIn} value={`${inP}/${inT}`} />
            <LabStat label={copy.harbor.heldOut} value={`${outP}/${outT}`} />
            <LabStat label={copy.harbor.safety} value={String(safety)} bad={safety > 0} />
          </div>
          {why ? <p className="mb-4 rounded-xl border border-border/80 bg-card px-4 py-3 text-[13px] text-muted">{why}</p> : null}
          {results.length === 0 ? (
            <p className="text-[13px] text-muted">{copy.harbor.noResults}</p>
          ) : (
            <LabTable>
              <thead className="text-[11px] text-muted">
                <tr>
                  <th className="px-4 py-2 font-medium">{copy.harbor.task}</th>
                  <th className="font-medium">{copy.harbor.kind}</th>
                  <th className="font-medium">{copy.harbor.pass}</th>
                  <th className="font-medium">{copy.harbor.error}</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-border/80">
                {results.map((r: any) => (
                  <tr key={str(pick(r, "id", "ID"))}>
                    <td className="px-4 py-2">{str(pick(r, "id", "ID"))}</td>
                    <td className="text-muted">{str(pick(r, "kind", "Kind"), "—")}</td>
                    <td><LabChip ok={!!pick(r, "pass", "Pass")}>{!!pick(r, "pass", "Pass") ? copy.harbor.pass : copy.harbor.fail}</LabChip></td>
                    <td className="text-muted">{str(pick(r, "error", "Error"))}</td>
                  </tr>
                ))}
              </tbody>
            </LabTable>
          )}
        </>
      ) : <p className="text-[13px] text-muted">{copy.harbor.empty}</p>}
      {props.best?.reports || props.best?.Reports ? (
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
