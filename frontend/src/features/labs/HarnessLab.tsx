import { useState } from "react";
import { pick, str } from "../../lib/normalize";
import { Button } from "../../components/ui/button";
import { Input } from "../../components/ui/input";
import { LabFrame } from "./LabFrame";
import { ConfirmDialog } from "../ConfirmDialog";

export function HarnessLab(props: {
  harness: any;
  diffA: string;
  diffB: string;
  diffOut: any;
  onA: (v: string) => void;
  onB: (v: string) => void;
  onCompare: () => void;
  onCheckout: (hash: string, l3?: boolean) => Promise<void> | void;
  onRollback: () => void;
}) {
  const refs = pick(props.harness, "refs") || {};
  const fields = (pick(props.diffOut, "fields") || []) as any[];
  const [pending, setPending] = useState<{ hash: string; surfaces: string } | null>(null);

  async function tryCheckout(hash: string) {
    try {
      await props.onCheckout(hash, false);
    } catch (e: any) {
      const msg = String(e?.message || e);
      const l3 = e?.body?.l3 || msg.includes("L3 confirmation");
      if (!l3) throw e;
      const surfaces = (e?.body?.surfaces || ["loop_preset/policy_pack"]).join(", ");
      setPending({ hash, surfaces });
    }
  }

  return (
    <LabFrame title="Harness" hint="Only pointers move. Checkout of loop/policy topology needs L3. Staging is not trusted.">
      <div className="mb-4">
        <Button variant="lift" onClick={props.onRollback}>Rollback</Button>
      </div>
      <div className="mb-6 grid gap-3 sm:grid-cols-2">
        {Object.entries(refs).map(([k, v]) => (
          <div className="rounded-2xl border border-border bg-panel p-4" key={k}>
            <div className="flex items-center justify-between">
              <div className="text-sm font-medium">{k}</div>
              {k === "staging" ? <span className="rounded-full bg-lift px-2 py-0.5 text-[11px] text-muted">not trusted</span> : null}
              {k === "active" ? <span className="rounded-full bg-accent/20 px-2 py-0.5 text-[11px] text-accent">trusted</span> : null}
            </div>
            <div className="mt-2 font-mono text-xs text-muted">{String(v).slice(0, 20)}</div>
            <div className="mt-3 flex gap-2">
              <Button size="sm" variant="lift" onClick={() => void tryCheckout(String(v))}>Checkout</Button>
              <Button size="sm" variant="ghost" onClick={() => props.onB(String(v))}>Diff as B</Button>
            </div>
          </div>
        ))}
      </div>
      <h3 className="mb-3 text-sm font-medium">Field diff</h3>
      <div className="mb-4 flex flex-wrap gap-2">
        <Input className="max-w-xs" placeholder="hash A" value={props.diffA} onChange={(e) => props.onA(e.target.value)} />
        <Input className="max-w-xs" placeholder="hash B" value={props.diffB} onChange={(e) => props.onB(e.target.value)} />
        <Button onClick={props.onCompare}>Compare</Button>
      </div>
      {fields.length ? (
        <table className="w-full text-left text-sm">
          <thead className="text-muted">
            <tr><th className="py-2">Field</th><th>From</th><th>To</th></tr>
          </thead>
          <tbody className="divide-y divide-border">
            {fields.map((f: any) => (
              <tr key={str(pick(f, "field", "Field"))} className={pick(f, "changed", "Changed") ? "bg-accent/5" : ""}>
                <td className="py-2">{str(pick(f, "field", "Field"))}</td>
                <td className="font-mono text-xs">{String(pick(f, "from", "From") || "").slice(0, 24)}</td>
                <td className="font-mono text-xs">{String(pick(f, "to", "To") || "").slice(0, 24)}</td>
              </tr>
            ))}
          </tbody>
        </table>
      ) : null}
      <ConfirmDialog
        open={!!pending}
        title="L3 checkout"
        danger
        confirmLabel="Checkout"
        body={pending ? `Changing ${pending.surfaces}. This is a human gate. Staging is not trusted.` : null}
        onCancel={() => setPending(null)}
        onConfirm={async () => {
          const hash = pending?.hash;
          setPending(null);
          if (hash) await props.onCheckout(hash, true);
        }}
      />
    </LabFrame>
  );
}
