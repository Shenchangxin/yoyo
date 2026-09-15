import { asArray, bool, num, pick, str } from "../../lib/normalize";
import { Button } from "../../components/ui/button";
import { Input } from "../../components/ui/input";
import { LabFrame } from "./LabFrame";

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
  const trials = asArray(pick(props.evolve, "tried", "Tried"));
  const bullets = asArray(pick(props.playbook, "bullets", "Bullets"));
  return (
    <LabFrame title="Evolve" hint="ACE deltas grow the playbook. Harbor still owns refs/active. Thumbs write refs/staging only.">
      <div className="mb-5 flex items-center gap-3">
        <label className="flex items-center gap-2 text-sm text-muted">
          K
          <Input type="number" min={1} max={8} className="w-16" value={props.k} onChange={(e) => props.onK(Number(e.target.value) || 3)} />
        </label>
        <Button disabled={props.busy} onClick={props.onRun}>Run cycle</Button>
      </div>
      {props.evolve ? (
        <div className="mb-5 grid grid-cols-3 gap-3">
          <Stat label="Promoted" value={slice(pick(props.evolve, "promoted", "Promoted") || "none")} />
          <Stat label="Parent" value={slice(pick(props.evolve, "parent_selected", "ParentSelected"))} />
          <Stat label="Trials" value={String(trials.length)} />
        </div>
      ) : <p className="mb-4 text-sm text-muted">No cycle yet.</p>}
      {trials.length ? (
        <table className="mb-6 w-full text-left text-sm">
          <thead className="text-muted">
            <tr><th className="py-2">Proposal</th><th>Accepted</th><th>Reason</th><th>In</th><th>Out</th></tr>
          </thead>
          <tbody className="divide-y divide-border">
            {trials.map((t: any, i: number) => {
              const m = pick(t, "metrics", "Metrics") || {};
              const p = pick(t, "proposal", "Proposal") || {};
              return (
                <tr key={str(pick(p, "id", "ID"), String(i))}>
                  <td className="py-2">{str(pick(p, "id", "ID"))}</td>
                  <td><Chip ok={bool(pick(t, "accepted", "Accepted"))}>{String(bool(pick(t, "accepted", "Accepted")))}</Chip></td>
                  <td className="text-muted">{str(pick(t, "reason", "Reason"))}</td>
                  <td>{num(pick(m, "held_in_pass", "HeldInPass"))}/{num(pick(m, "held_in_total", "HeldInTotal"))}</td>
                  <td>{num(pick(m, "held_out_pass", "HeldOutPass"))}/{num(pick(m, "held_out_total", "HeldOutTotal"))}</td>
                </tr>
              );
            })}
          </tbody>
        </table>
      ) : null}
      <h3 className="mb-3 text-sm font-medium">Playbook</h3>
      <div className="mb-6 grid gap-3 sm:grid-cols-2">
        {bullets.map((b: any) => (
          <div className="rounded-2xl border border-border bg-panel p-4" key={str(pick(b, "id", "ID"))}>
            <div className="text-sm">{str(pick(b, "text", "Text"))}</div>
            <div className="mt-1 text-xs text-muted">+{num(pick(b, "helpful", "Helpful"))} / −{num(pick(b, "harmful", "Harmful"))}</div>
            <div className="mt-3 flex gap-2">
              <Button size="sm" variant="lift" onClick={() => props.onRate(str(pick(b, "id", "ID")), true)}>Helpful</Button>
              <Button size="sm" variant="lift" onClick={() => props.onRate(str(pick(b, "id", "ID")), false)}>Harmful</Button>
            </div>
          </div>
        ))}
      </div>
      <h3 className="mb-3 text-sm font-medium">Archive</h3>
      <div className="grid gap-2 sm:grid-cols-3">
        {props.archive.map((n: any) => (
          <div className="rounded-2xl border border-border bg-panel p-3" key={str(pick(n, "id", "ID"))}>
            <div className="font-mono text-xs">{slice(pick(n, "id", "ID"))}</div>
            <div className="mt-1 text-xs text-muted">{str(pick(n, "proposal_id", "ProposalID", "note", "Note"))}</div>
            <div className="mt-1 text-xs">{bool(pick(n, "accepted", "Accepted")) ? "accepted" : "rejected"}</div>
          </div>
        ))}
      </div>
    </LabFrame>
  );
}

function Stat({ label, value }: { label: string; value: string }) {
  return (
    <div className="rounded-2xl border border-border bg-panel p-4">
      <div className="text-xs text-muted">{label}</div>
      <div className="mt-1 truncate font-mono text-sm font-semibold">{value}</div>
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

function slice(v: any): string {
  const s = str(v);
  return s.length > 16 ? s.slice(0, 16) : s || "—";
}
