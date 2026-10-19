import { useEffect } from "react";
import { asArray, bool, num, pick, str } from "../../lib/normalize";
import { formatMoney, useCopy } from "../../lib/i18n";
import { cn } from "../../lib/utils";
import { Button } from "../../components/ui/button";
import { Input } from "../../components/ui/input";
import { LabCard, LabChip, LabFrame, LabTable } from "./LabFrame";
import { PresenceModuleLoading } from "../presence";
import { useUI } from "../../lib/store";

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
  overlay?: any;
  overlayK?: number;
  onOverlayK?: (n: number) => void;
  overlayMode?: string;
  onOverlayMode?: (mode: string) => void;
  overlayLibrary?: any;
  overlayReuse?: boolean;
  onOverlayReuse?: (v: boolean) => void;
  overlayDisabled?: boolean;
  overlayDisabledReason?: string;
  onOverlay?: () => void;
  onOverlayProve?: (hash: string) => void;
  evolve: any;
  playbook: any;
  archive: any[];
  onRate: (id: string, helpful: boolean) => void;
}) {
  const copy = useCopy();
  const lab = useUI((s) => s.showHarnessLab);
  const overlayK = Math.max(1, Math.min(4, props.overlayK || 1));
  useEffect(() => {
    if (!props.busy) return;
    useUI.getState().setModuleLoading(true);
    return () => useUI.getState().setModuleLoading(false);
  }, [props.busy]);
  if (props.busy) {
    return (
      <LabFrame>
        <PresenceModuleLoading />
      </LabFrame>
    );
  }
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
  const overlayHint = copy.labs.overlayHint
    .replace("{r}", String(4 * overlayK))
    .replace("{p}", String(3 * overlayK))
    .replace("{k}", String(overlayK));
  return (
    <LabFrame>
      <section className="mb-8" data-testid="overlay-thread">
        <div className="flex flex-wrap items-start justify-between gap-3">
          <div className="min-w-0">
            <h3 className="text-[15px] font-medium tracking-[-0.02em]">{copy.labs.overlayTitle}</h3>
            <p className="mt-1 max-w-[52ch] text-[13px] text-muted">{overlayHint}</p>
          </div>
          <Button
            disabled={props.busy || !!props.overlayDisabled}
            title={props.overlayDisabled ? props.overlayDisabledReason : undefined}
            onClick={props.onOverlay}
          >
            {copy.labs.overlayRun}
          </Button>
        </div>
        <div className="mt-[var(--space-group)] flex flex-col gap-[var(--space-group)]">
          <Gate on={!!props.overlayReuse} onClick={() => props.onOverlayReuse?.(!props.overlayReuse)}>{copy.labs.overlayReuse}</Gate>
          {lab ? (
            <details>
              <summary className="fold-summary">{copy.labs.showAdvanced}</summary>
              <div className="mt-[var(--space-group)] flex flex-col gap-[var(--space-group)]">
                <label className="flex flex-col gap-[var(--space-item)] text-[13px] font-medium text-foreground">
                  {copy.labs.overlayK}
                  <Input type="number" min={1} max={4} className="h-8 w-full font-normal" value={overlayK} onChange={(e) => props.onOverlayK?.(Math.max(1, Math.min(4, Number(e.target.value) || 1)))} />
                </label>
                <div className="flex flex-wrap gap-2">
                  <Gate on={(props.overlayMode || "agg") !== "seq"} onClick={() => props.onOverlayMode?.("agg")}>{copy.labs.overlayWidth}</Gate>
                  <Gate on={props.overlayMode === "seq"} onClick={() => props.onOverlayMode?.("seq")}>{copy.labs.overlayDepth}</Gate>
                </div>
              </div>
            </details>
          ) : null}
        </div>
        {props.overlay ? <OverlayResult overlay={props.overlay} copy={copy} lab={lab} onProve={props.onOverlayProve} /> : null}
        <LibraryBlock view={props.overlayLibrary} copy={copy} />
      </section>

      <section className="mb-8 border-t border-border/70 pt-6">
        <div className="flex flex-wrap items-start justify-between gap-3">
          <div className="min-w-0">
            <h3 className="text-[15px] font-medium tracking-[-0.02em]">{copy.labs.agentTitle}</h3>
            <p className="mt-1 max-w-[52ch] text-[13px] text-muted">{copy.labs.agentHint}</p>
          </div>
          <Button disabled={props.busy} onClick={props.onRun}>{copy.labs.runCycle}</Button>
        </div>
        <details className="mt-[var(--space-group)]">
          <summary className="fold-summary">{copy.labs.showAdvanced}</summary>
          <div className="mt-[var(--space-group)] flex max-w-sm flex-col gap-[var(--space-group)]">
            <label className="flex flex-col gap-[var(--space-item)] text-[13px] font-medium text-foreground">
              {copy.labs.k}
              <Input type="number" min={1} max={8} className="h-8 w-full font-normal" value={props.k} onChange={(e) => props.onK(Number(e.target.value) || 3)} />
            </label>
            <label className="flex flex-col gap-[var(--space-item)] text-[13px] font-medium text-foreground">
              {copy.labs.rounds}
              <Input type="number" min={1} max={20} className="h-8 w-full font-normal" value={props.rounds} onChange={(e) => props.onRounds(Number(e.target.value) || 1)} />
            </label>
            {lab ? (
              <div className="flex flex-wrap gap-2">
                <Gate on={props.sealed} onClick={() => props.onSealed(!props.sealed)}>{copy.labs.sealed}</Gate>
                <Gate on={props.behavior} onClick={() => props.onBehavior(!props.behavior)}>{copy.labs.behavior}</Gate>
                <Gate on={props.index} onClick={() => props.onIndex(!props.index)}>{copy.labs.index}</Gate>
                <Gate on={props.baselines} onClick={() => props.onBaselines(!props.baselines)}>{copy.labs.baselines}</Gate>
              </div>
            ) : null}
            <label className="flex flex-col gap-[var(--space-item)] text-[13px] font-medium text-foreground">
              {copy.labs.maxUsd}
              <Input type="number" min={0} step={0.1} className="h-8 w-full font-normal" value={props.maxUsd || ""} onChange={(e) => props.onMaxUsd(Number(e.target.value) || 0)} />
            </label>
          </div>
        </details>
        {props.evolve ? <CycleResult evolve={props.evolve} copy={copy} lab={lab} /> : <p className="mt-4 text-[13px] text-muted">{copy.labs.noCycle}</p>}
      </section>

      <LabSpendTable evolve={props.evolve} overlay={props.overlay} copy={copy} lab={lab} />
      {lab ? <p className="mb-5 text-[13px] text-muted">{copy.labs.goNoGo}</p> : null}
      {deltas.length ? (
        <div className="mb-5 overflow-hidden rounded-md bg-sidebar/40">
          <div className="border-b border-border/70 px-4 py-2 text-[12px] font-medium">{copy.labs.thisCycle} · {copy.labs.playbookDelta}</div>
          {deltas.map((d, i) => (
            <div key={d.id || i} className={cn("px-4 py-2.5 text-[13px]", i > 0 && "border-t border-border/70")}>{d.text}</div>
          ))}
        </div>
      ) : props.evolve ? <p className="mb-5 text-[13px] text-muted">{copy.labs.noDelta}</p> : null}
      {lab && trials.length ? (
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
      {lab ? <CompareBlock evolve={props.evolve} copy={copy} /> : null}
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
          <div className="overflow-hidden rounded-md bg-sidebar/40">
            {props.archive.map((n: any, i: number) => (
              <div key={str(pick(n, "id", "ID"), String(i))} className={cn("flex gap-3 px-4 py-3", i > 0 && "border-t border-border/70")}>
                <div className="mt-1 size-1.5 shrink-0 rounded-full bg-foreground/70" aria-hidden />
                <div className="min-w-0 flex-1">
                  <div className="flex flex-wrap items-center gap-2">
                    {lab ? <span className="font-mono text-[12px]">{slice(pick(n, "id", "ID"))}</span> : null}
                    <span className="text-[13px]">{bool(pick(n, "accepted", "Accepted")) ? copy.labs.accepted : copy.labs.rejected}</span>
                  </div>
                  <div className="mt-0.5 text-[12px] text-muted">
                    {str(pick(n, "proposal_id", "ProposalID", "note", "Note")) || copy.labs.parent}
                    {lab && str(pick(n, "parent", "Parent")) ? ` · ${slice(pick(n, "parent", "Parent"))}` : ""}
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

function CycleResult(props: { evolve: any; copy: ReturnType<typeof useCopy>; lab: boolean }) {
  const moved = bool(pick(props.evolve, "active_moved", "ActiveMoved"));
  const merged = bool(pick(props.evolve, "merged", "Merged"));
  const line = moved ? props.copy.labs.cycleSwitched : merged ? props.copy.labs.cycleMadeCandidate : props.copy.labs.cycleKept;
  if (!props.lab) {
    return <p className="mt-4 text-[13px] text-foreground">{line}</p>;
  }
  return (
    <div className="mt-4 grid grid-cols-3 gap-3">
      <div className="px-0.5 py-2">
        <div className="text-[11px] text-muted">{props.copy.labs.promoted}</div>
        <div className="mt-1 truncate font-mono text-[13px]">{slice(pick(props.evolve, "promoted", "Promoted") || "none")}</div>
      </div>
      <div className="px-0.5 py-2">
        <div className="text-[11px] text-muted">{props.copy.labs.merged}</div>
        <div className="mt-1 text-[15px] font-medium">{merged ? props.copy.labs.accepted : props.copy.labs.rejected}</div>
      </div>
      <div className="px-0.5 py-2">
        <div className="text-[11px] text-muted">{props.copy.labs.activeMoved}</div>
        <div className="mt-1 text-[15px] font-medium">{moved ? props.copy.rsi.active : props.copy.rsi.canary}</div>
      </div>
    </div>
  );
}

function slice(v: any): string {
  const s = str(v);
  return s.length > 16 ? s.slice(0, 16) : s || "—";
}

function LibraryBlock(props: { view?: any; copy: ReturnType<typeof useCopy> }) {
  const copy = props.copy;
  const entries = asArray(pick(props.view, "entries", "Entries"));
  const domain = str(pick(props.view, "domain", "Domain"));
  return (
    <div className="mt-4" data-testid="overlay-library">
      <h4 className="text-[12px] font-medium">{copy.labs.overlayLibraryTitle}{domain ? ` · ${domain}` : ""}</h4>
      {entries.length === 0 ? (
        <p className="mt-1 text-[12px] text-muted">{copy.labs.overlayLibraryEmpty}</p>
      ) : (
        <ul className="mt-2 space-y-2">
          {entries.map((e: any, i: number) => (
            <li key={str(pick(e, "id", "ID"), String(i))} className="rounded-md bg-sidebar/40 px-3 py-2 text-[12px]">
              <div className="font-medium">{str(pick(e, "cue", "Cue")) || "—"}</div>
              <div className="mt-1 text-muted">{str(pick(e, "lesson", "Lesson"))}</div>
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}

function OverlayResult(props: { overlay: any; copy: ReturnType<typeof useCopy>; lab: boolean; onProve?: (hash: string) => void }) {
  const adopted = bool(pick(props.overlay, "adopted", "Adopted"));
  const hash = str(pick(props.overlay, "hash", "Hash"));
  const r = num(pick(props.overlay, "r", "R"));
  const p = num(pick(props.overlay, "p", "P"));
  const lib = str(pick(props.overlay, "library", "LibraryStatus"));
  const fell = asArray(pick(props.overlay, "fell_back", "FellBack"));
  const next = adopted ? props.copy.labs.overlayPin : props.copy.labs.overlayKeep;
  const libLine = lib === "added" ? props.copy.labs.overlayLibrary : props.copy.labs.overlayNoEntry;
  const ledger = props.copy.labs.overlayLedger.replace("{r}", String(r)).replace("{p}", String(p));
  return (
    <div className="mt-4 rounded-md bg-sidebar/40 px-4 py-3" data-testid="overlay-next">
      <p className="text-[15px] font-medium leading-snug tracking-[-0.02em] text-pretty text-foreground">{next}</p>
      <p className="mt-1 text-[12px] text-muted">
        {ledger} · {libLine}
        {props.lab && hash ? ` · ${hash.slice(0, 8)}` : ""}
      </p>
      {fell.length ? <p className="mt-1 text-[12px] text-muted">{props.copy.labs.overlayFellBack}</p> : null}
      {adopted && hash && props.onProve ? (
        <div className="mt-3">
          <Button size="sm" variant="lift" title={props.copy.labs.overlayProveHint} onClick={() => props.onProve?.(hash)}>
            {props.copy.labs.overlayProve}
          </Button>
        </div>
      ) : null}
    </div>
  );
}

function LabSpendTable(props: { evolve: any; overlay?: any; copy: ReturnType<typeof useCopy>; lab: boolean }) {
  const copy = props.copy;
  const cmp = pick(props.evolve, "compare", "Compare");
  const overlay = props.overlay;
  const rows: { name: string; identity: string; usd: number; pass: string }[] = [];
  if (props.evolve) {
    const s = pick(props.evolve, "spend", "Spend") || {};
    rows.push({
      name: copy.labs.agentTitle,
      identity: "—",
      usd: num(pick(s, "usd", "USD")),
      pass: "—",
    });
  }
  if (props.lab && cmp) {
    asArray(pick(cmp, "best_of_n", "BestOfN")).forEach((r: any, i: number) => {
      const m = pick(r, "metrics", "Metrics") || {};
      rows.push({
        name: `${copy.labs.bon} ${i + 1}`,
        identity: "—",
        usd: num(pick(r, "usd", "USD")),
        pass: `${num(pick(m, "held_in_pass", "HeldInPass"))}/${num(pick(m, "held_in_total", "HeldInTotal"))} · ${num(pick(m, "held_out_pass", "HeldOutPass"))}/${num(pick(m, "held_out_total", "HeldOutTotal"))}`,
      });
    });
    asArray(pick(cmp, "iid", "IID")).forEach((t: any, i: number) => {
      const m = pick(t, "metrics", "Metrics") || {};
      rows.push({
        name: `${copy.labs.iid} ${i + 1}`,
        identity: "—",
        usd: num(pick(pick(t, "spend", "Spend") || {}, "usd", "USD")),
        pass: `${num(pick(m, "held_in_pass", "HeldInPass"))}/${num(pick(m, "held_in_total", "HeldInTotal"))} · ${num(pick(m, "held_out_pass", "HeldOutPass"))}/${num(pick(m, "held_out_total", "HeldOutTotal"))}`,
      });
    });
    asArray(pick(cmp, "scs", "SCS")).forEach((t: any, i: number) => {
      const m = pick(t, "metrics", "Metrics") || {};
      rows.push({
        name: `${copy.labs.scs} ${i + 1}`,
        identity: "—",
        usd: num(pick(pick(t, "spend", "Spend") || {}, "usd", "USD")),
        pass: `${num(pick(m, "held_in_pass", "HeldInPass"))}/${num(pick(m, "held_in_total", "HeldInTotal"))} · ${num(pick(m, "held_out_pass", "HeldOutPass"))}/${num(pick(m, "held_out_total", "HeldOutTotal"))}`,
      });
    });
  }
  if (overlay) {
    const r = num(pick(overlay, "r", "R"));
    const p = num(pick(overlay, "p", "P"));
    const k = num(pick(overlay, "k", "K")) || 1;
    const s = pick(overlay, "spend", "Spend") || {};
    rows.push({
      name: copy.labs.overlayRow,
      identity: copy.labs.overlayLedger.replace("{r}", String(r)).replace("{p}", String(p)) + (props.lab ? ` · k ${k}` : ""),
      usd: num(pick(s, "usd", "USD")),
      pass: "—",
    });
  }
  if (!rows.length) return null;
  if (!props.lab) {
    return (
      <p className="mb-5 text-[13px] text-muted" data-testid="lab-spend">
        {rows.map((row) => `${row.name}${row.identity !== "—" ? ` · ${row.identity}` : ""} · ${formatMoney(row.usd)}`).join(" · ")}
      </p>
    );
  }
  return (
    <div className="mb-5" data-testid="lab-spend">
      <h3 className="mb-3 text-[13px] font-medium">{copy.labs.labSpend}</h3>
      <LabTable>
        <thead className="text-[11px] text-muted">
          <tr>
            <th className="px-4 py-2 font-medium">{copy.labs.labSpend}</th>
            <th className="font-medium">{copy.labs.identity}</th>
            <th className="font-medium">{copy.labs.usd}</th>
            <th className="font-medium">{copy.labs.in}/{copy.labs.out}</th>
          </tr>
        </thead>
        <tbody className="divide-y divide-border/80">
          {rows.map((row, i) => (
            <tr key={`${row.name}-${i}`}>
              <td className="px-4 py-2">{row.name}</td>
              <td className="tabular-nums text-muted">{row.identity}</td>
              <td className="tabular-nums">{formatMoney(row.usd)}</td>
              <td className="tabular-nums text-muted">{row.pass}</td>
            </tr>
          ))}
        </tbody>
      </LabTable>
    </div>
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
