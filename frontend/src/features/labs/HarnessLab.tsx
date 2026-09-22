import { useEffect, useState } from "react";
import { pick, str } from "../../lib/normalize";
import { useCopy } from "../../lib/i18n";
import { Button } from "../../components/ui/button";
import { Input } from "../../components/ui/input";
import { LabFrame, LabTable } from "./LabFrame";
import { ConfirmDialog } from "../ConfirmDialog";
import { parseHarnessRefs, parseMaterialChanges, shortHash } from "../../lib/harness-refs";
import { MaterialChangeList } from "../harness/Changes";

export function HarnessLab(props: {
  harness: any;
  diffA: string;
  diffB: string;
  diffOut: any;
  onA: (v: string) => void;
  onB: (v: string) => void;
  onCompare: (a?: string, b?: string) => void;
  onCheckout: (hash: string, l3?: boolean) => Promise<void> | void;
  onRollback: () => void;
}) {
  const copy = useCopy();
  const refs = parseHarnessRefs(props.harness);
  const fields = (pick(props.diffOut, "fields", "Fields") || []) as any[];
  const changes = parseMaterialChanges(pick(props.diffOut, "changes", "Changes"));
  const [pending, setPending] = useState<{ hash: string; surfaces: string } | null>(null);
  const [advanced, setAdvanced] = useState(false);
  const [showHashes, setShowHashes] = useState(false);

  useEffect(() => {
    if (refs.active && refs.staging) {
      props.onA(refs.active);
      props.onB(refs.staging);
      props.onCompare(refs.active, refs.staging);
    }
    // once when both hashes exist
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [refs.active, refs.staging]);

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

  const pointers = [
    { id: "active", hash: refs.active, hint: copy.labs.trusted },
    { id: "staging", hash: refs.staging, hint: copy.labs.notTrusted },
    { id: "canary", hash: refs.canary, hint: copy.rsi.canary },
  ].filter((p) => p.hash);

  return (
    <LabFrame>
      <p className="mb-4 max-w-[62ch] text-[13px] leading-5 text-muted">{copy.labs.ceremony}</p>
      <div className="mb-6 overflow-hidden rounded-md bg-sidebar/40">
        {pointers.map((p, i) => (
          <div key={p.id} className={"flex items-center gap-3 px-4 py-3" + (i > 0 ? " border-t border-border/70" : "")}>
            <div className="min-w-0 flex-1">
              <div className="text-[13px] font-medium">{p.id}</div>
              <div className="font-mono text-[11px] text-muted" title={p.hash}>{shortHash(p.hash, 12)}</div>
            </div>
            <span className="text-[11px] text-muted">{p.hint}</span>
            {p.id !== "active" ? (
              <Button size="sm" onClick={() => void tryCheckout(p.hash)}>{copy.labs.moveActive}</Button>
            ) : null}
          </div>
        ))}
      </div>
      {refs.active && refs.staging ? (
        <div className="mb-4">
          <Button variant="lift" onClick={() => props.onCompare(refs.active, refs.staging)}>{copy.labs.compareStaging}</Button>
        </div>
      ) : null}
      <h3 className="mb-3 text-[13px] font-medium">{copy.labs.fieldDiff}</h3>
      {props.diffOut ? (
        <div className="mb-4">
          <MaterialChangeList changes={changes} empty={copy.rsi.noChanges} />
        </div>
      ) : null}
      {fields.length ? (
        <button type="button" className="mb-3 text-[12px] text-muted hover:text-foreground" onClick={() => setShowHashes((v) => !v)}>
          {copy.labs.hashFields}
        </button>
      ) : null}
      {showHashes && fields.length ? (
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
              <tr key={str(pick(f, "field", "Field"))} className={pick(f, "changed", "Changed") ? "bg-lift/50" : ""}>
                <td className="px-4 py-2">{str(pick(f, "field", "Field"))}</td>
                <td className="font-mono text-[11px]">{String(pick(f, "from", "From") || "").slice(0, 24)}</td>
                <td className="font-mono text-[11px]">{String(pick(f, "to", "To") || "").slice(0, 24)}</td>
              </tr>
            ))}
          </tbody>
        </LabTable>
      ) : null}
      <button type="button" className="mt-4 text-[12px] text-muted hover:text-foreground" onClick={() => setAdvanced((v) => !v)}>
        {copy.labs.advancedHashes}
      </button>
      {advanced ? (
        <div className="mt-3 mb-4 flex flex-wrap gap-2">
          <Input className="h-8 max-w-xs" placeholder={copy.labs.hashA} value={props.diffA} onChange={(e) => props.onA(e.target.value)} />
          <Input className="h-8 max-w-xs" placeholder={copy.labs.hashB} value={props.diffB} onChange={(e) => props.onB(e.target.value)} />
          <Button variant="lift" onClick={() => props.onCompare()}>{copy.labs.compare}</Button>
        </div>
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
