import { asArray, bool, num, pick, str } from "../../lib/normalize";
import { useCopy } from "../../lib/i18n";
import { Button } from "../../components/ui/button";
import { Input } from "../../components/ui/input";
import { LabCard, LabChip, LabFrame, LabStat, LabTable } from "./LabFrame";

export function EvolveLab(props: {
  busy: boolean;
  k: number;
  onK: (n: number) => void;
  onRun: () => void;
  evolve: any;
  playbook: any;
  archive: any[];
  onRate: (id: string, helpful: boolean) => void;
}) {
  const copy = useCopy();
  const trials = asArray(pick(props.evolve, "tried", "Tried"));
  const bullets = asArray(pick(props.playbook, "bullets", "Bullets"));
  return (
    <LabFrame>
      <div className="mb-5 flex items-center gap-3">
        <label className="flex items-center gap-2 text-[13px] text-muted">
          {copy.labs.k}
          <Input type="number" min={1} max={8} className="h-8 w-16" value={props.k} onChange={(e) => props.onK(Number(e.target.value) || 3)} />
        </label>
        <Button disabled={props.busy} onClick={props.onRun}>{copy.labs.runCycle}</Button>
      </div>
      {props.evolve ? (
        <div className="mb-5 grid grid-cols-3 gap-3">
          <LabStat label={copy.labs.promoted} value={slice(pick(props.evolve, "promoted", "Promoted") || "none")} />
          <LabStat label={copy.labs.parent} value={slice(pick(props.evolve, "parent_selected", "ParentSelected"))} />
          <LabStat label={copy.labs.trials} value={String(trials.length)} />
        </div>
      ) : <p className="mb-4 text-[13px] text-muted">{copy.labs.noCycle}</p>}
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
      <h3 className="mb-3 text-[13px] font-medium">{copy.labs.archive}</h3>
      <div className="grid gap-2 sm:grid-cols-3">
        {props.archive.map((n: any) => (
          <LabCard className="p-3" key={str(pick(n, "id", "ID"))}>
            <div className="font-mono text-[11px]">{slice(pick(n, "id", "ID"))}</div>
            <div className="mt-1 text-[11px] text-muted">{str(pick(n, "proposal_id", "ProposalID", "note", "Note"))}</div>
            <div className="mt-1 text-[11px]">{bool(pick(n, "accepted", "Accepted")) ? copy.labs.accepted : copy.labs.rejected}</div>
          </LabCard>
        ))}
      </div>
    </LabFrame>
  );
}

function slice(v: any): string {
  const s = str(v);
  return s.length > 16 ? s.slice(0, 16) : s || "—";
}
