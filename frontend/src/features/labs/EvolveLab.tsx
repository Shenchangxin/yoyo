import { asArray, bool, num, pick, str } from "../../lib/normalize";
import { useCopy } from "../../lib/i18n";
import { cn } from "../../lib/utils";
import { Button } from "../../components/ui/button";
import { Input } from "../../components/ui/input";
import { LabCard, LabChip, LabFrame, LabStat, LabStrip, LabTable } from "./LabFrame";

export function EvolveLab(props: {
  busy: boolean;
  k: number;
  onK: (n: number) => void;
  rounds: number;
  onRounds: (n: number) => void;
  sealed: boolean;
  onSealed: (v: boolean) => void;
  behavior: boolean;
  onBehavior: (v: boolean) => void;
  index: boolean;
  onIndex: (v: boolean) => void;
  baselines: boolean;
  onBaselines: (v: boolean) => void;
  maxUsd: number;
  onMaxUsd: (n: number) => void;
  onRun: () => void;
  evolve: any;
  playbook: any;
  archive: any[];
  onRate: (id: string, helpful: boolean) => void;
}) {
  const copy = useCopy();
  const trials = asArray(pick(props.evolve, "tried", "Tried"));
  const bullets = asArray(pick(props.playbook, "bullets", "Bullets"));
  const deltas = trials.flatMap((t: any) => {
    if (!bool(pick(t, "accepted", "Accepted"))) return [];
    const p = pick(t, "proposal", "Proposal") || {};
    const b = pick(p, "playbook_bullet", "PlaybookBullet");
    const text = str(pick(b, "text", "Text") || pick(p, "expected", "Expected"));
    if (!text) return [];
    return [{ id: str(pick(p, "id", "ID")), text }];
  });
  return (
    <LabFrame>
      <LabStrip>
        <label className="flex items-center gap-2 text-[12.5px] text-muted">
          {copy.labs.k}
          <Input type="number" min={1} max={8} className="h-8 w-16" value={props.k} onChange={(e) => props.onK(Number(e.target.value) || 3)} />
        </label>
        <label className="flex items-center gap-2 text-[12.5px] text-muted">
          {copy.labs.rounds}
          <Input type="number" min={1} max={20} className="h-8 w-16" value={props.rounds} onChange={(e) => props.onRounds(Number(e.target.value) || 1)} />
        </label>
        <Gate on={props.sealed} onClick={() => props.onSealed(!props.sealed)}>{copy.labs.sealed}</Gate>
        <Gate on={props.behavior} onClick={() => props.onBehavior(!props.behavior)}>{copy.labs.behavior}</Gate>
        <Gate on={props.index} onClick={() => props.onIndex(!props.index)}>{copy.labs.index}</Gate>
        <Gate on={props.baselines} onClick={() => props.onBaselines(!props.baselines)}>{copy.labs.baselines}</Gate>
        <label className="flex items-center gap-2 text-[12.5px] text-muted">
          {copy.labs.maxUsd}
          <Input type="number" min={0} step={0.1} className="h-8 w-20" value={props.maxUsd || ""} onChange={(e) => props.onMaxUsd(Number(e.target.value) || 0)} />
        </label>
        <Button className="ml-auto" disabled={props.busy} onClick={props.onRun}>{copy.labs.runCycle}</Button>
      </LabStrip>
      {props.evolve ? (
        <div className="mb-5 grid grid-cols-3 gap-3">
          <LabStat label={copy.labs.promoted} value={slice(pick(props.evolve, "promoted", "Promoted") || "none")} />
          <LabStat label={copy.labs.merged} value={bool(pick(props.evolve, "merged", "Merged")) ? copy.labs.accepted : copy.labs.rejected} />
          <LabStat label={copy.labs.activeMoved} value={bool(pick(props.evolve, "active_moved", "ActiveMoved")) ? copy.rsi.active : copy.rsi.canary} />
        </div>
      ) : <p className="mb-4 text-[13px] text-muted">{copy.labs.noCycle}</p>}
      {props.evolve ? <SpendBlock spend={pick(props.evolve, "spend", "Spend")} copy={copy} /> : null}
      <p className="mb-5 text-[12.5px] text-muted">{copy.labs.goNoGo}</p>
      {deltas.length ? (
        <div className="mb-5 overflow-hidden rounded-[10px] border border-border/80 bg-card">
          <div className="border-b border-border/70 px-4 py-2 text-[12px] font-medium">{copy.labs.thisCycle} · {copy.labs.playbookDelta}</div>
          {deltas.map((d, i) => (
            <div key={d.id || i} className={cn("px-4 py-2.5 text-[13px]", i > 0 && "border-t border-border/70")}>{d.text}</div>
          ))}
        </div>
      ) : props.evolve ? <p className="mb-5 text-[12.5px] text-muted">{copy.labs.noDelta}</p> : null}
      {trials.length ? (
        <div className="mb-6">
          <LabTable>
            <thead className="text-[11px] text-muted">
              <tr>
                <th className="px-4 py-2 font-medium">{copy.labs.proposal}</th>
                <th className="font-medium">{copy.labs.accepted}</th>
                <th className="font-medium">{copy.labs.reason}</th>
                <th className="font-medium">{copy.labs.in}</th>
                <th className="font-medium">{copy.labs.out}</th>
                <th className="font-medium">{copy.labs.hit}</th>
                <th className="font-medium">{copy.labs.miss}</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-border/80">
              {trials.map((t: any, i: number) => {
                const m = pick(t, "metrics", "Metrics") || {};
                const p = pick(t, "proposal", "Proposal") || {};
                return (
                  <tr key={str(pick(p, "id", "ID"), String(i))}>
                    <td className="px-4 py-2">{str(pick(p, "id", "ID"))}</td>
                    <td><LabChip ok={bool(pick(t, "accepted", "Accepted"))}>{bool(pick(t, "accepted", "Accepted")) ? copy.labs.accepted : copy.labs.rejected}</LabChip></td>
                    <td className="text-muted">{str(pick(t, "reason", "Reason"))}</td>
                    <td className="tabular-nums">{num(pick(m, "held_in_pass", "HeldInPass"))}/{num(pick(m, "held_in_total", "HeldInTotal"))}</td>
                    <td className="tabular-nums">{num(pick(m, "held_out_pass", "HeldOutPass"))}/{num(pick(m, "held_out_total", "HeldOutTotal"))}</td>
                    <td className="tabular-nums">{num(pick(t, "manifesto_hit", "ManifestoHit"))}</td>
                    <td className="tabular-nums">{num(pick(t, "manifesto_miss", "ManifestoMiss"))}</td>
                  </tr>
                );
              })}
            </tbody>
          </LabTable>
        </div>
      ) : null}
      <CompareBlock evolve={props.evolve} copy={copy} />
      <h3 className="mb-3 text-[13px] font-medium">{copy.labs.playbook}</h3>
      <div className="mb-6 grid gap-3 sm:grid-cols-2">
        {bullets.map((b: any) => (
          <LabCard key={str(pick(b, "id", "ID"))}>
            <div className="text-[13px]">{str(pick(b, "text", "Text"))}</div>
            <div className="mt-1 text-[11px] text-muted">+{num(pick(b, "helpful", "Helpful"))} / −{num(pick(b, "harmful", "Harmful"))}</div>
            <div className="mt-3 flex gap-2">
              <Button size="sm" variant="lift" onClick={() => props.onRate(str(pick(b, "id", "ID")), true)}>{copy.labs.helpful}</Button>
              <Button size="sm" variant="lift" onClick={() => props.onRate(str(pick(b, "id", "ID")), false)}>{copy.labs.harmful}</Button>
            </div>
          </LabCard>
        ))}
      </div>
      <h3 className="mb-3 text-[13px] font-medium">{copy.labs.timeline}</h3>
      <div className="relative mb-2">
        {props.archive.length === 0 ? (
          <p className="text-[13px] text-muted">{copy.labs.noCycle}</p>
        ) : (
          <div className="overflow-hidden rounded-[10px] border border-border/80 bg-card">
            {props.archive.map((n: any, i: number) => (
              <div key={str(pick(n, "id", "ID"), String(i))} className={cn("flex gap-3 px-4 py-3", i > 0 && "border-t border-border/70")}>
                <div className="mt-1 size-1.5 shrink-0 rounded-full bg-foreground/70" aria-hidden />
                <div className="min-w-0 flex-1">
                  <div className="flex flex-wrap items-center gap-2">
                    <span className="font-mono text-[12px]">{slice(pick(n, "id", "ID"))}</span>
                    <span className="text-[11px] text-muted">{bool(pick(n, "accepted", "Accepted")) ? copy.labs.accepted : copy.labs.rejected}</span>
                  </div>
                  <div className="mt-0.5 text-[12px] text-muted">
                    {str(pick(n, "proposal_id", "ProposalID", "note", "Note")) || copy.labs.parent}
                    {str(pick(n, "parent", "Parent")) ? ` · ${slice(pick(n, "parent", "Parent"))}` : ""}
                  </div>
                </div>
              </div>
            ))}
          </div>
        )}
      </div>
    </LabFrame>
  );
}

function slice(v: any): string {
  const s = str(v);
  return s.length > 16 ? s.slice(0, 16) : s || "—";
}

function SpendBlock(props: { spend: any; copy: ReturnType<typeof useCopy> }) {
  const s = props.spend || {};
  const usd = num(pick(s, "usd", "USD"));
  const tin = num(pick(s, "tokens_in", "TokensIn"));
  const tout = num(pick(s, "tokens_out", "TokensOut"));
  const wall = num(pick(s, "wall_ms", "WallMs"));
  if (!usd && !tin && !tout && !wall) return null;
  return (
    <p className="mb-4 text-[12.5px] text-muted">
      {props.copy.labs.spend} · {props.copy.labs.usd} {usd.toFixed(4)} · {props.copy.labs.tokens} {tin}/{tout} · {props.copy.labs.wall} {wall}ms
    </p>
  );
}

function CompareBlock(props: { evolve: any; copy: ReturnType<typeof useCopy> }) {
  const cmp = pick(props.evolve, "compare", "Compare");
  if (!cmp) return null;
  const copy = props.copy;
  const rows = [
    { name: copy.labs.bon, reports: asArray(pick(cmp, "best_of_n", "BestOfN")) },
    { name: copy.labs.iid, trials: asArray(pick(cmp, "iid", "IID")) },
    { name: copy.labs.scs, trials: asArray(pick(cmp, "scs", "SCS")) },
  ];
  return (
    <div className="mb-6">
      <h3 className="mb-3 text-[13px] font-medium">{copy.labs.labCompare}</h3>
      <p className="mb-3 font-mono text-[12px] text-muted">
        {slice(pick(cmp, "snapshot", "Snapshot"))} · {str(pick(cmp, "model_fingerprint", "ModelFingerprint")) || "—"}
      </p>
      <SpendBlock spend={pick(cmp, "spend", "Spend")} copy={copy} />
      <LabTable>
        <thead className="text-[11px] text-muted">
          <tr>
            <th className="px-4 py-2 font-medium">{copy.labs.labCompare}</th>
            <th className="font-medium">{copy.labs.in}</th>
            <th className="font-medium">{copy.labs.out}</th>
          </tr>
        </thead>
        <tbody className="divide-y divide-border/80">
          {rows.flatMap((row) => {
            if (row.reports) {
              return row.reports.map((r: any, i: number) => {
                const m = pick(r, "metrics", "Metrics") || {};
                return (
                  <tr key={`${row.name}-${i}`}>
                    <td className="px-4 py-2">{row.name} {i + 1}</td>
                    <td className="tabular-nums">{num(pick(m, "held_in_pass", "HeldInPass"))}/{num(pick(m, "held_in_total", "HeldInTotal"))}</td>
                    <td className="tabular-nums">{num(pick(m, "held_out_pass", "HeldOutPass"))}/{num(pick(m, "held_out_total", "HeldOutTotal"))}</td>
                  </tr>
                );
              });
            }
            return (row.trials || []).map((t: any, i: number) => {
              const m = pick(t, "metrics", "Metrics") || {};
              return (
                <tr key={`${row.name}-${i}`}>
                  <td className="px-4 py-2">{row.name} {str(pick(pick(t, "proposal", "Proposal") || {}, "id", "ID"), String(i + 1))}</td>
                  <td className="tabular-nums">{num(pick(m, "held_in_pass", "HeldInPass"))}/{num(pick(m, "held_in_total", "HeldInTotal"))}</td>
                  <td className="tabular-nums">{num(pick(m, "held_out_pass", "HeldOutPass"))}/{num(pick(m, "held_out_total", "HeldOutTotal"))}</td>
                </tr>
              );
            });
          })}
        </tbody>
      </LabTable>
    </div>
  );
}

function Gate(props: { on: boolean; onClick: () => void; children: string }) {
  return (
    <button
      type="button"
      aria-pressed={props.on}
      className={cn(
        "h-8 cursor-pointer rounded-md px-2.5 text-[12px] font-medium transition-colors",
        props.on ? "bg-lift text-foreground" : "text-muted hover:bg-lift/50 hover:text-foreground",
      )}
      onClick={props.onClick}
    >
      {props.children}
    </button>
  );
}
