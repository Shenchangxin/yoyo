import { Gamepad2 } from "lucide-react";
import { Button } from "../../components/ui/button";
import { EmptyState } from "../../components/ui/empty-state";
import { cn } from "../../lib/utils";
import { GAME_CAMERAS, qaStatus, type GameCamera, type GameRow } from "./game-model";
import { GameInspector } from "./GameInspector";
import { buildMarks, pageLead, type GameDeskCopy } from "./game-desk";

type DeskCopy = GameDeskCopy & {
  nameLabel: string;
  titleHint: string;
  intentLabel: string;
  intentHint: string;
  camera: string;
  tps: string;
  fps: string;
  top: string;
  saveIntent: string;
  saveAndAsk: string;
  askNow: string;
  buildWait: string;
  buildReady: string;
  goPlay: string;
  noPlayable: string;
  noPlayableHint: string;
  making: string;
  objective: string;
  win: string;
  fail: string;
  elements: string;
  interactions: string;
  raw: string;
  jobs: string;
  collect: string;
  spec: string;
  qa: string;
  qaPass: string;
  qaFail: string;
  qaPending: string;
};

export function GoalPage(props: {
  copy: DeskCopy;
  title: string;
  intent: string;
  camera: GameCamera;
  busy: boolean;
  onTitle: (v: string) => void;
  onIntent: (v: string) => void;
  onCamera: (v: GameCamera) => void;
  onSave: () => void;
  onAsk: () => void;
}) {
  const g = props.copy;
  const cameraLabel = (id: string) => id === "fps" ? g.fps : id === "top" ? g.top : g.tps;
  return (
    <div className="mx-auto flex h-full w-full max-w-[min(100%,36rem)] flex-col overflow-auto px-5 py-6 sm:px-8" data-testid="game-view-intent">
      <h2 className="text-[18px] font-semibold tracking-[-0.03em]">{g.stepIntent}</h2>
      <p className="mt-1.5 text-[13px] leading-[1.55] text-muted">{pageLead("intent", g)}</p>
      <label className="mt-6 block text-[11px] font-medium text-muted" htmlFor="game-title-field">{g.nameLabel}</label>
      <input
        id="game-title-field"
        data-testid="game-title-field"
        className="mt-1.5 w-full rounded-lg bg-lift px-2.5 py-1.5 text-[13px] text-foreground outline-none ring-1 ring-transparent focus:ring-accent"
        placeholder={g.titleHint}
        value={props.title}
        onChange={(e) => props.onTitle(e.target.value)}
      />
      <label className="mt-3 block text-[11px] font-medium text-muted" htmlFor="game-intent-field">{g.intentLabel}</label>
      <textarea
        id="game-intent-field"
        data-testid="game-intent-field"
        rows={4}
        className="mt-1.5 w-full resize-none rounded-lg bg-lift px-2.5 py-2 text-[13px] text-foreground outline-none ring-1 ring-transparent focus:ring-accent"
        placeholder={g.intentHint}
        value={props.intent}
        onChange={(e) => props.onIntent(e.target.value)}
      />
      <div className="mt-3 flex flex-wrap items-center gap-1.5">
        <span className="mr-1 text-[11px] text-muted">{g.camera}</span>
        {GAME_CAMERAS.map((id) => (
          <button
            key={id}
            type="button"
            aria-pressed={props.camera === id}
            className={cn(
              "rounded-md px-2 py-1 text-[12px]",
              props.camera === id ? "bg-lift text-foreground" : "text-muted hover:bg-lift/70 hover:text-foreground",
            )}
            onClick={() => props.onCamera(id)}
          >
            {cameraLabel(id)}
          </button>
        ))}
      </div>
      <div className="mt-6 flex flex-wrap gap-2">
        <Button size="sm" disabled={props.busy} data-testid="game-ask-intent" onClick={props.onAsk}>{g.saveAndAsk}</Button>
        <Button size="sm" variant="ghost" disabled={props.busy} onClick={props.onSave}>{g.saveIntent}</Button>
      </div>
    </div>
  );
}

export function MakePage(props: {
  copy: DeskCopy;
  row: GameRow;
  busy: boolean;
  onAsk: () => void;
  onMakeNow: () => void;
  onPlay: () => void;
}) {
  const g = props.copy;
  const marks = buildMarks(props.row.stage, g);
  const playable = marks.some((m) => m.key === "play" && m.done);
  return (
    <div className="mx-auto flex h-full w-full max-w-[min(100%,36rem)] flex-col overflow-auto px-5 py-6 sm:px-8" data-testid="game-view-build">
      <h2 className="text-[18px] font-semibold tracking-[-0.03em]">{g.stepBuild}</h2>
      <p className="mt-1.5 text-[13px] leading-[1.55] text-muted">{pageLead("build", g)}</p>
      <ol className="mt-6 space-y-2">
        {marks.map((m) => (
          <li key={m.key} className="flex items-center gap-2 text-[13px]">
            <span className={cn("size-1.5 rounded-full", m.done ? "bg-accent" : "bg-border")} aria-hidden />
            <span className={m.done ? "text-foreground" : "text-muted"}>{m.label}</span>
          </li>
        ))}
      </ol>
      <p className="mt-4 text-[13px] text-muted">{props.busy ? g.buildWait : playable ? g.buildReady : g.noPlayableHint}</p>
      <div className="mt-6 flex flex-wrap gap-2">
        {playable ? (
          <Button size="sm" data-testid="game-go-play" onClick={props.onPlay}>{g.goPlay}</Button>
        ) : (
          <Button size="sm" disabled={props.busy} data-testid="game-ask-build" onClick={props.onAsk}>{g.askBuild}</Button>
        )}
        <Button size="sm" variant="ghost" disabled={props.busy} data-testid="game-make-now" onClick={props.onMakeNow}>{g.askNow}</Button>
      </div>
    </div>
  );
}

export function ReviewPage(props: {
  copy: DeskCopy;
  row: GameRow;
  busy: boolean;
  onAsk: () => void;
}) {
  const g = props.copy;
  const qa = qaStatus(props.row);
  const qaLabel = qa === "pass" ? g.qaPass : qa === "fail" ? g.qaFail : qa === "pending" ? g.qaPending : "";
  const cameraLabel = (id: string) => id === "fps" ? g.fps : id === "top" ? g.top : g.tps;
  return (
    <div className="flex h-full min-h-0 flex-col overflow-hidden" data-testid="game-view-check">
      <div className="mx-auto w-full max-w-[min(100%,36rem)] shrink-0 px-5 pt-6 sm:px-8">
        <h2 className="text-[18px] font-semibold tracking-[-0.03em]">{g.stepCheck}</h2>
        <p className="mt-1.5 text-[13px] leading-[1.55] text-muted">{pageLead("check", g)}</p>
        {qaLabel ? (
          <p className="mt-3 text-[13px]" data-testid="game-qa-summary">{g.qa} · {qaLabel}</p>
        ) : null}
        <div className="mt-4 flex flex-wrap gap-2">
          <Button size="sm" disabled={props.busy} data-testid="game-ask-check" onClick={props.onAsk}>
            {qa === "fail" ? g.askRepair : g.askCheck}
          </Button>
        </div>
      </div>
      <div className="mt-4 min-h-0 flex-1 overflow-hidden border-t border-border/60">
        <GameInspector
          row={props.row}
          copy={{
            objective: g.objective,
            win: g.win,
            fail: g.fail,
            elements: g.elements,
            interactions: g.interactions,
            raw: g.raw,
            jobs: g.jobs,
            collect: g.collect,
            cameraLabel,
            qa: g.qa,
            qaPass: g.qaPass,
            qaFail: g.qaFail,
            qaPending: g.qaPending,
          }}
        />
      </div>
    </div>
  );
}

export function PlayEmpty(props: {
  copy: DeskCopy;
  busy: boolean;
  onAsk: () => void;
}) {
  const g = props.copy;
  return (
    <EmptyState
      className="h-full"
      icon={<Gamepad2 className="size-5" />}
      title={props.busy ? g.making : g.noPlayable}
      body={g.noPlayableHint}
      action={<Button size="sm" disabled={props.busy} onClick={props.onAsk}>{g.askBuild}</Button>}
    />
  );
}
