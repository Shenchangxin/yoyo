import { useState } from "react";
import { pick, str } from "../../lib/normalize";
import { useCopy } from "../../lib/i18n";
import { Button } from "../../components/ui/button";
import { Input } from "../../components/ui/input";
import { LabCard, LabFrame, LabTable } from "./LabFrame";
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
  const copy = useCopy();
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
    <LabFrame>
      <div className="mb-6 grid gap-3 sm:grid-cols-2">
        {Object.entries(refs).map(([k, v]) => (
          <LabCard key={k}>
            <div className="flex items-center justify-between">
              <div className="text-[13px] font-medium">{k}</div>
              {k === "staging" ? <span className="rounded-full bg-lift px-2 py-0.5 text-[11px] text-muted">{copy.labs.notTrusted}</span> : null}
              {k === "active" ? <span className="rounded-full bg-accent/15 px-2 py-0.5 text-[11px] text-accent">{copy.labs.trusted}</span> : null}
            </div>
            <div className="mt-2 font-mono text-[11px] text-muted">{String(v).slice(0, 20)}</div>
            <div className="mt-3 flex gap-2">
              <Button size="sm" onClick={() => void tryCheckout(String(v))}>{copy.labs.checkout}</Button>
              <Button size="sm" variant="ghost" onClick={() => props.onB(String(v))}>{copy.labs.diffAsB}</Button>
            </div>
          </LabCard>
        ))}
      </div>
      <h3 className="mb-3 text-[13px] font-medium">{copy.labs.fieldDiff}</h3>
      <div className="mb-4 flex flex-wrap gap-2">
        <Input className="h-8 max-w-xs" placeholder={copy.labs.hashA} value={props.diffA} onChange={(e) => props.onA(e.target.value)} />
        <Input className="h-8 max-w-xs" placeholder={copy.labs.hashB} value={props.diffB} onChange={(e) => props.onB(e.target.value)} />
        <Button variant="lift" onClick={props.onCompare}>{copy.labs.compare}</Button>
      </div>
      {fields.length ? (
        <LabTable>
          <thead className="text-[11px] text-muted">
            <tr>
              <th className="px-4 py-2 font-medium">{copy.labs.field}</th>
              <th className="font-medium">{copy.labs.from}</th>
              <th className="font-medium">{copy.labs.to}</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-border/80">
            {fields.map((f: any) => (
              <tr key={str(pick(f, "field", "Field"))} className={pick(f, "changed", "Changed") ? "bg-accent/5" : ""}>
                <td className="px-4 py-2">{str(pick(f, "field", "Field"))}</td>
                <td className="font-mono text-[11px]">{String(pick(f, "from", "From") || "").slice(0, 24)}</td>
                <td className="font-mono text-[11px]">{String(pick(f, "to", "To") || "").slice(0, 24)}</td>
              </tr>
            ))}
          </tbody>
        </LabTable>
      ) : null}
      <div className="mt-8 border-t border-border/80 pt-4">
        <div className="mb-2 text-[11px] font-medium text-muted">{copy.rsi.danger}</div>
        <Button variant="danger" onClick={props.onRollback}>{copy.labs.rollback}</Button>
      </div>
      <ConfirmDialog
        open={!!pending}
        title={copy.labs.l3}
        danger
        confirmLabel={copy.labs.checkout}
        body={pending ? copy.settings.checkoutConfirm : null}
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
