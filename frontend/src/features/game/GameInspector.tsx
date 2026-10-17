import { Badge } from "../../components/ui/badge";
import { cn } from "../../lib/utils";
import { inspectGame, qaStatus, type GameRow } from "./game-model";

export function GameInspector(props: {
  row: GameRow;
  className?: string;
  copy: {
    objective: string;
    win: string;
    fail: string;
    elements: string;
    interactions: string;
    raw: string;
    jobs: string;
    collect: string;
    cameraLabel: (id: string) => string;
    qa: string;
    qaPass: string;
    qaFail: string;
    qaPending: string;
  };
}) {
  const info = inspectGame(props.row);
  const qa = qaStatus(props.row);
  const checks = props.row.verification?.checks || {};
  const raw = JSON.stringify(props.row.ir || props.row.playable_spec || {}, null, 2);
  const qaLabel = qa === "pass" ? props.copy.qaPass : qa === "fail" ? props.copy.qaFail : qa === "pending" ? props.copy.qaPending : "";

  return (
    <aside className={cn("flex min-h-0 w-full min-w-0 flex-1 flex-col overflow-auto", props.className)} data-testid="game-inspector">
      <div className="flex flex-wrap gap-1.5 border-b border-border/50 px-3 py-2">
        <Badge>{props.copy.cameraLabel(info.camera)}</Badge>
        {info.need > 0 ? <Badge>{props.copy.collect} {info.need}</Badge> : null}
        {qaLabel ? <Badge data-testid="game-qa-badge">{props.copy.qa} {qaLabel}</Badge> : null}
        {info.jobs.length ? <Badge>{props.copy.jobs} {info.jobs.length}</Badge> : null}
      </div>
      {Object.keys(checks).length ? (
        <List
          label={props.copy.qa}
          rows={Object.entries(checks).map(([key, value]) => ({ key, title: key, meta: String(value) }))}
        />
      ) : null}
      <Section label={props.copy.objective} body={info.objective || props.row.intent} />
      <Section label={props.copy.win} body={info.win} />
      <Section label={props.copy.fail} body={info.fail} />
      {info.elements.length ? (
        <List label={props.copy.elements} rows={info.elements.map((el) => ({ key: el.id, meta: el.type, title: el.id }))} />
      ) : null}
      {info.interactions.length ? (
        <List label={props.copy.interactions} rows={info.interactions.map((it) => ({ key: it.id, meta: it.trigger, title: it.id }))} />
      ) : null}
      {props.row.error ? (
        <Section label="" body={props.row.error} danger />
      ) : null}
      <details className="border-t border-border/50 px-3 py-2">
        <summary className="cursor-pointer text-[11px] font-medium text-muted">{props.copy.raw}</summary>
        <pre className="mt-2 overflow-auto font-mono text-[11px] leading-[1.55] text-muted">{raw}</pre>
      </details>
    </aside>
  );
}

function Section(props: { label: string; body?: string; danger?: boolean }) {
  if (!props.body) return null;
  return (
    <div className="border-b border-border/50 px-3 py-2.5">
      {props.label ? <p className="text-[11px] font-medium text-muted">{props.label}</p> : null}
      <p className={cn("text-[13px] leading-[1.5] text-pretty", props.danger ? "text-danger" : "text-foreground", props.label && "mt-1")}>
        {props.body}
      </p>
    </div>
  );
}

function List(props: { label: string; rows: { key: string; title: string; meta: string }[] }) {
  return (
    <div className="border-b border-border/50 px-3 py-2.5">
      <p className="text-[11px] font-medium text-muted">{props.label}</p>
      <ul className="mt-1.5 space-y-1">
        {props.rows.map((row) => (
          <li key={row.key} className="flex items-baseline justify-between gap-2 text-[12px]">
            <span className="truncate font-mono text-[11.5px]">{row.title}</span>
            <span className="shrink-0 text-[11px] text-muted">{row.meta}</span>
          </li>
        ))}
      </ul>
    </div>
  );
}
