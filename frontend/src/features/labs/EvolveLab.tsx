import { asArray, bool, num, pick, str } from "../../lib/normalize";
import { useCopy } from "../../lib/i18n";
import { cn } from "../../lib/utils";
import { Button } from "../../components/ui/button";
import { Input } from "../../components/ui/input";
import { LabCard, LabChip, LabFrame, LabStat, LabTable } from "./LabFrame";

export function EvolveLab(props: {
  busy: boolean;
  k: number;
  onK: (n: number) => void;
  rounds: number;
  onRounds: (n: number) => void;
  sealed: boolean;
  onSealed: (v: boolean) => void;
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
      <div className="mb-5 flex flex-wrap items-center gap-3">
        <label className="flex items-center gap-2 text-[13px] text-muted">
          {copy.labs.k}
          <Input type="number" min={1} max={8} className="h-8 w-16" value={props.k} onChange={(e) => props.onK(Number(e.target.value) || 3)} />
        </label>
        <label className="flex items-center gap-2 text-[13px] text-muted">
          {copy.labs.rounds}
          <Input type="number" min={1} max={20} className="h-8 w-16" value={props.rounds} onChange={(e) => props.onRounds(Number(e.target.value) || 1)} />
        </label>
        <label className="flex items-center gap-2 text-[13px] text-muted">
          <input type="checkbox" checked={props.sealed} onChange={(e) => props.onSealed(e.target.checked)} />
          {copy.labs.sealed}
        </label>
        <Button disabled={props.busy} onClick={props.onRun}>{copy.labs.runCycle}</Button>
      </div>
      {props.evolve ? (
        <div className="mb-5 grid grid-cols-3 gap-3">
          <LabStat label={copy.labs.promoted} value={slice(pick(props.evolve, "promoted", "Promoted") || "none")} />
          <LabStat label={copy.labs.merged} value={bool(pick(props.evolve, "merged", "Merged")) ? copy.labs.accepted : copy.labs.rejected} />
          <LabStat label={copy.labs.activeMoved} value={bool(pick(props.evolve, "active_moved", "ActiveMoved")) ? copy.rsi.active : copy.rsi.canary} />
        </div>
      ) : <p className="mb-4 text-[13px] text-muted">{copy.labs.noCycle}</p>}
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
                  </tr>
                );
              })}
            </tbody>
          </LabTable>
        </div>
      ) : null}
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
