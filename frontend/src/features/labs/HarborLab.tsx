import { asArray, num, pick, str } from "../../lib/normalize";
import { useCopy } from "../../lib/i18n";
import { Button } from "../../components/ui/button";
import { Input } from "../../components/ui/input";
import { LabFrame } from "./LabFrame";

export function HarborLab(props: {
  busy: string | null;
  error?: string;
  lastKind?: "suite" | "safety" | "tb" | "bon" | "models";
  report: any;
  best: any;
  models: string;
  onModels: (v: string) => void;
  onRun: (kind: "suite" | "safety" | "tb" | "bon" | "models") => void;
}) {
  const copy = useCopy();
  const metrics = pick(props.report, "metrics", "Metrics") || {};
  const results = asArray(pick(props.report, "results", "Results"));
  const inP = num(pick(metrics, "held_in_pass", "HeldInPass"));
  const inT = num(pick(metrics, "held_in_total", "HeldInTotal"));
  const outP = num(pick(metrics, "held_out_pass", "HeldOutPass"));
  const outT = num(pick(metrics, "held_out_total", "HeldOutTotal"));
  const safety = num(pick(metrics, "safety_fail", "SafetyFail"));
  const why =
    safety > 0
      ? "Safety suite failed — Harbor will not promote."
      : props.report
        ? "This run is evidence. Promotion still requires a candidate vs baseline non-regression gate."
        : "";

  return (
    <LabFrame title="Harbor" hint="Held-in must not drop, held-out must not drop, safety fail never promotes.">
      <div className="mb-5 flex flex-wrap items-center gap-2">
        <Button disabled={!!props.busy} onClick={() => props.onRun("suite")}>Run suite</Button>
        <Button variant="lift" disabled={!!props.busy} onClick={() => props.onRun("safety")}>Suite + safety</Button>
        <Button variant="lift" disabled={!!props.busy} onClick={() => props.onRun("tb")}>TB subset</Button>
        <Button variant="lift" disabled={!!props.busy} onClick={() => props.onRun("bon")}>Best-of-3</Button>
        <Input className="max-w-xs" placeholder="models a,b" value={props.models} onChange={(e) => props.onModels(e.target.value)} />
        <Button variant="lift" disabled={!!props.busy} onClick={() => props.onRun("models")}>Models BoN</Button>
      </div>
      {props.busy ? <p className="mb-4 text-sm text-accent">{props.busy}</p> : null}
      {!props.busy && props.error ? (
        <div className="mb-4 flex items-center gap-3 rounded-2xl border border-danger/30 bg-danger/10 px-4 py-3 text-sm text-danger">
          <span>{props.error}</span>
          <Button size="sm" variant="lift" onClick={() => props.onRun(props.lastKind || "suite")}>{copy.harbor.retry}</Button>
        </div>
      ) : null}
      {props.report ? (
        <>
          <div className="mb-4 grid grid-cols-3 gap-3">
            <Stat label={copy.harbor.heldIn} value={`${inP}/${inT}`} />
            <Stat label={copy.harbor.heldOut} value={`${outP}/${outT}`} />
            <Stat label={copy.harbor.safety} value={String(safety)} bad={safety > 0} />
          </div>
          {why ? <p className="mb-4 rounded-2xl border border-border bg-panel px-4 py-3 text-sm text-muted">{why}</p> : null}
          {results.length === 0 ? (
            <p className="text-sm text-muted">{copy.harbor.noResults}</p>
          ) : (
          <table className="w-full text-left text-sm">
            <thead className="text-muted">
              <tr><th className="py-2">Task</th><th>Pass</th><th>Error</th></tr>
            </thead>
            <tbody className="divide-y divide-border">
              {results.map((r: any) => (
                <tr key={str(pick(r, "id", "ID"))}>
                  <td className="py-2">{str(pick(r, "id", "ID"))}</td>
                  <td><Chip ok={!!pick(r, "pass", "Pass")}>{String(!!pick(r, "pass", "Pass"))}</Chip></td>
                  <td className="text-muted">{str(pick(r, "error", "Error"))}</td>
                </tr>
              ))}
            </tbody>
          </table>
          )}
        </>
      ) : <p className="text-sm text-muted">{copy.harbor.empty}</p>}
      {props.best?.reports || props.best?.Reports ? (
        <div className="mt-4 grid gap-3 sm:grid-cols-2">
          {asArray(pick(props.best, "reports", "Reports")).map((r: any, i: number) => {
            const m = pick(r, "metrics", "Metrics") || {};
            return (
              <div className="rounded-2xl border border-border bg-panel p-4" key={i}>
                <div className="text-sm font-medium">Run {i + 1}</div>
                <div className="mt-1 text-xs text-muted">
                  in {num(pick(m, "held_in_pass", "HeldInPass"))}/{num(pick(m, "held_in_total", "HeldInTotal"))} · out {num(pick(m, "held_out_pass", "HeldOutPass"))}/{num(pick(m, "held_out_total", "HeldOutTotal"))}
                </div>
              </div>
            );
          })}
        </div>
      ) : null}
    </LabFrame>
  );
}

function Stat({ label, value, bad }: { label: string; value: string; bad?: boolean }) {
  return (
    <div className="rounded-2xl border border-border bg-panel p-4">
      <div className="text-xs text-muted">{label}</div>
      <div className={"mt-1 text-xl font-semibold " + (bad ? "text-danger" : "")}>{value}</div>
    </div>
  );
}

function Chip({ ok, children }: { ok: boolean; children: string }) {
  return (
    <span className={"rounded-full px-2 py-0.5 text-xs " + (ok ? "bg-accent/20 text-accent" : "bg-danger/15 text-danger")}>
      {children}
    </span>
  );
}
