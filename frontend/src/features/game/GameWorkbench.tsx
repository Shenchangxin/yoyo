import { useCallback, useEffect, useRef, useState } from "react";
import { ArrowLeft, Check, ChevronDown, Gamepad2, MoreHorizontal, Plus } from "lucide-react";
import { motion } from "motion/react";
import { toast } from "sonner";
import { Button } from "../../components/ui/button";
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuLabel, DropdownMenuSeparator, DropdownMenuTrigger } from "../../components/ui/dropdown-menu";
import { EmptyState } from "../../components/ui/empty-state";
import { Kbd } from "../../components/ui/kbd";
import { ProgressHairline } from "../../components/ui/progress-hairline";
import { cn } from "../../lib/utils";
import { useCopy } from "../../lib/i18n";
import { useUI } from "../../lib/store";
import * as api from "../../lib/client";
import { DURATION, motionTransition, useMotionReduced } from "../../lib/motion";
import { SandboxedFrame } from "../transcript/SandboxedFrame";
import { ConfirmDialog } from "../ConfirmDialog";
import { useGameSelection } from "../video/workshop-store";
import type { VideoProject } from "../../lib/protocol";
import { createGameWorld } from "./game-actions";
import { GoalPage, MakePage, PlayEmpty, ReviewPage } from "./GameDeskViews";
import { GAME_PAGES, askAgentLabel, askAgentText, deskPage, pageComplete, pageLabel } from "./game-desk";
import {
  hasPlayable,
  normalizeCamera,
  stageRank,
  type GameCamera,
  type GameRow,
  type GameStep,
} from "./game-model";

function asList<T>(v: unknown): T[] {
  return Array.isArray(v) ? (v as T[]) : [];
}

export function GameWorkbench(props: {
  sessionId?: string;
  projects?: VideoProject[];
  onNeedSession: () => void | Promise<string | void>;
  onOpenProject?: (p: VideoProject) => void;
  onRefreshHistory?: () => void;
  onAskAgent?: (text: string) => void;
}) {
  const copy = useCopy();
  const g = copy.video.gameDesk;
  const reduced = useMotionReduced();
  const rail = motionTransition(reduced, DURATION);
  const gameId = useGameSelection((s) => s.gameId);
  const focus = useGameSelection((s) => s.focus);
  const rev = useGameSelection((s) => s.rev);
  const setGameId = useGameSelection((s) => s.setGameId);
  const setGameFocus = useGameSelection((s) => s.setGameFocus);
  const pendingPlay = useRef(false);
  const [rows, setRows] = useState<GameRow[]>([]);
  const [current, setCurrent] = useState<GameRow | null>(null);
  const [intent, setIntent] = useState("");
  const [title, setTitle] = useState("");
  const [camera, setCamera] = useState<GameCamera>("tps");
  const [busy, setBusy] = useState("");
  const [switchOpen, setSwitchOpen] = useState(false);
  const [ready, setReady] = useState(!gameId);
  const [loadErr, setLoadErr] = useState("");
  const [kill, setKill] = useState<GameRow | null>(null);

  const page = deskPage(current, focus);
  const playable = hasPlayable(current);

  const loadList = useCallback(async () => {
    setRows(asList<GameRow>(await api.video.listGames()));
  }, []);

  const loadCurrent = useCallback(async (id: string) => {
    const got = (await api.video.getGame(id, props.sessionId || "")) as GameRow;
    setCurrent(got);
    if (got?.id) useGameSelection.getState().setGameId(got.id, got.title || "");
    return got;
  }, [props.sessionId]);

  useEffect(() => {
    void loadList().catch((e) => toast.error(api.errMessage(e)));
  }, [loadList]);

  useEffect(() => {
    if (!current) return;
    setIntent(current.intent || "");
    setTitle(current.title || "");
    setCamera(normalizeCamera(current.camera));
    if (!useGameSelection.getState().focus) {
      setGameFocus(deskPage(current, ""));
    }
  }, [current?.id, setGameFocus]);

  useEffect(() => {
    if (!gameId) {
      setCurrent(null);
      setLoadErr("");
      setReady(true);
      return;
    }
    const id = gameId;
    if (!current || current.id !== id) setReady(false);
    setLoadErr("");
    void loadCurrent(id).then((got) => {
      if (useGameSelection.getState().gameId === id) setReady(true);
      if (pendingPlay.current && hasPlayable(got)) {
        pendingPlay.current = false;
        setGameFocus("play");
      }
    }).catch((e) => {
      if (useGameSelection.getState().gameId !== id) return;
      setLoadErr(api.errMessage(e));
      setReady(true);
    });
  }, [gameId, rev, loadCurrent, setGameFocus]);

  async function ensureSession() {
    if (props.sessionId) return props.sessionId;
    const id = await props.onNeedSession();
    return typeof id === "string" ? id : "";
  }

  async function run(label: string, fn: () => Promise<void>) {
    setBusy(label);
    try {
      await fn();
      await loadList();
      if (useGameSelection.getState().gameId) {
        await loadCurrent(useGameSelection.getState().gameId).catch(() => undefined);
      }
      props.onRefreshHistory?.();
    } catch (e) {
      toast.error(api.errMessage(e));
    } finally {
      setBusy("");
    }
  }

  async function openWorld(row: GameRow) {
    setGameFocus("");
    setGameId(row.id, row.title);
    setCurrent(row);
    props.onOpenProject?.({ kind: "game", id: row.id, title: row.title || copy.video.newGame });
    const got = await api.video.getGame(row.id).catch(() => row);
    setCurrent(got as GameRow);
  }

  async function pipeline(row: GameRow) {
    const sessionId = await ensureSession();
    await api.video.updateGame({
      id: row.id,
      session_id: sessionId,
      intent: intent.trim() || row.intent,
      camera,
      title: title.trim() || row.title,
    }).catch(() => undefined);
    const rank = stageRank(row.stage);
    const voice = intent.trim() || row.intent || "";
    if (rank < 1) await api.video.analyzeGame(row.id, voice, sessionId);
    if (rank < 2) await api.video.planGameRules(row.id, voice, sessionId);
    if (rank < 3) await api.video.extractGameCandidates(row.id, sessionId);
    if (rank < 4) await api.video.placeGame(row.id, sessionId);
    if (rank < 5) await api.video.realizeGameAssets(row.id, sessionId);
    await api.video.compileGame(row.id, sessionId);
    setGameFocus("play");
  }

  async function finishBuild(row: GameRow) {
    await run("build", () => pipeline(row));
  }

  function goHome() {
    setGameFocus("");
    setGameId("");
    setCurrent(null);
    setIntent("");
    setTitle("");
    setCamera("tps");
  }

  async function createWorld() {
    await run("create", async () => {
      const created = await createGameWorld({ title: copy.video.newGame });
      setGameFocus("intent");
      props.onOpenProject?.({ kind: "game", id: created.id, title: created.title || copy.video.newGame });
    });
  }

  async function saveIntent() {
    if (!current) return false;
    const voice = intent.trim();
    if (!voice) {
      toast.error(g.needIntent);
      return false;
    }
    await run("intent", async () => {
      const sessionId = await ensureSession();
      await api.video.updateGame({
        id: current.id,
        session_id: sessionId,
        intent: voice,
        title: title.trim() || current.title,
        camera,
      });
    });
    return true;
  }

  function playNow() {
    const path = current?.playable || "";
    if (!path) return;
    useUI.getState().setReviewFile(path);
    useUI.getState().setInspTab("browser");
    useUI.getState().setInspector(true);
  }

  function ask(step: GameStep = page) {
    if (!current) return;
    const row = { ...current, intent: intent.trim() || current.intent, title: title.trim() || current.title, camera };
    props.onAskAgent?.(askAgentText(step, row, g));
  }

  const statusLabel = (stage?: string) => {
    if (stage === "scene") return g.statusScene;
    if (stage === "rules") return g.statusRules;
    if (stage === "placed" || stage === "candidates") return g.statusPlaced;
    if (stage === "assets") return g.statusAssets;
    if (stage === "compiled") return g.statusCompiled;
    if (stage === "verified") return g.statusVerified;
    if (stage === "failed") return g.statusFailed;
    return g.statusDraft;
  };

  return (
    <div className="@container flex h-full min-h-0 flex-col" data-testid="game-workbench" data-bound={current ? "true" : "false"} data-page={page}>
      {busy ? <ProgressHairline indeterminate /> : null}
      <header className="flex shrink-0 items-center gap-2 border-b border-border/70 px-3 py-2">
        <div className="min-w-0 flex-1">
          <div className="flex min-w-0 items-center gap-0.5">
            {current ? (
              <Button size="sm" variant="ghost" className="h-7 w-7 shrink-0 px-0" data-testid="game-back-home" title={g.back} aria-label={g.back} onClick={goHome}>
                <ArrowLeft className="size-3.5" aria-hidden />
              </Button>
            ) : null}
            <h1 className="min-w-0 truncate text-[14px] font-semibold tracking-[-0.03em]">
              {current?.title || copy.video.shellGame}
            </h1>
            <DropdownMenu open={switchOpen} onOpenChange={setSwitchOpen}>
              <DropdownMenuTrigger asChild>
                <Button size="sm" variant="ghost" className="h-6 w-6 shrink-0 px-0" aria-label={g.switch} title={g.switch} data-testid="game-switch">
                  <ChevronDown className="size-3.5 text-muted" aria-hidden />
                </Button>
              </DropdownMenuTrigger>
              <DropdownMenuContent align="start" className="w-80 max-h-80 overflow-y-auto">
                <DropdownMenuItem disabled={!!busy} data-testid="game-switch-new" onSelect={() => { void createWorld(); }}>
                  <Plus className="size-3.5 shrink-0 opacity-70" aria-hidden />
                  {copy.video.newGame}
                </DropdownMenuItem>
                <DropdownMenuItem data-testid="game-switch-home" onSelect={goHome}>
                  <ArrowLeft className="size-3.5 shrink-0 opacity-70" aria-hidden />
                  {g.askAgent}
                </DropdownMenuItem>
                <DropdownMenuSeparator />
                <DropdownMenuLabel>{g.worlds}</DropdownMenuLabel>
                {rows.length ? rows.map((row) => (
                  <DropdownMenuItem key={row.id} data-testid={`game-row-${row.id}`} onSelect={() => { void openWorld(row); }}>
                    <span className="min-w-0 flex-1">
                      <span className="block truncate text-[13px]">{row.title || copy.video.newGame}</span>
                      <span className="mt-0.5 block truncate text-[11px] text-muted">{statusLabel(row.stage)}</span>
                    </span>
                    {row.id === gameId ? <Check className="size-3.5 shrink-0" aria-hidden /> : null}
                  </DropdownMenuItem>
                )) : (
                  <DropdownMenuItem disabled>{copy.video.gameEmpty}</DropdownMenuItem>
                )}
              </DropdownMenuContent>
            </DropdownMenu>
          </div>
          <p className="truncate text-[12px] text-muted">
            {current ? [pageLabel(page, g), current.intent || g.homeHint].filter(Boolean).join(" · ") : g.homeHint}
          </p>
        </div>
        {current ? (
          <Button size="sm" className="h-7 shrink-0 text-[12px]" disabled={!!busy} data-testid="game-ask" onClick={() => ask()}>
            {askAgentLabel(page, current, g)}
          </Button>
        ) : null}
        {current && page === "play" && playable ? (
          <Button size="sm" variant="ghost" className="h-7 shrink-0 text-[12px]" data-testid="game-open-browser" onClick={playNow}>
            {g.openBrowser}
          </Button>
        ) : null}
        <Button
          size="sm"
          variant="ghost"
          className="h-7 shrink-0 gap-1 px-2 text-[12px]"
          disabled={!!busy}
          data-testid="game-new"
          title={copy.video.newGame}
          onClick={() => void createWorld()}
        >
          <Plus className="size-3.5" aria-hidden />
          <span className="hidden @[420px]:inline">{copy.video.newGame}</span>
        </Button>
        {current ? (
          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <Button size="sm" variant="ghost" className="h-7 w-7 shrink-0 px-0" data-testid="game-more" aria-label={g.more}>
                <MoreHorizontal className="size-3.5" aria-hidden />
              </Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end">
              <DropdownMenuItem disabled={!!busy} onSelect={() => ask("build")}>{g.askBuild}</DropdownMenuItem>
              <DropdownMenuItem className="text-danger" onSelect={() => setKill(current)}>{g.delete}</DropdownMenuItem>
            </DropdownMenuContent>
          </DropdownMenu>
        ) : null}
      </header>

      {current ? (
        <nav className="flex shrink-0 items-center gap-1 border-b border-border/70 px-2 py-1.5" aria-label={copy.video.shellGame} data-testid="game-pipeline">
          {GAME_PAGES.map((step, i) => {
            const on = page === step;
            const done = pageComplete(step, current);
            return (
              <button
                key={step}
                type="button"
                data-testid={`game-step-${step}`}
                aria-current={on ? "page" : undefined}
                className={cn(
                  "relative inline-flex items-center gap-1 rounded-md px-2 py-1 text-[12px] font-medium",
                  on ? "text-foreground" : done ? "text-foreground/80 hover:bg-lift/70" : "text-muted hover:bg-lift/70 hover:text-foreground",
                )}
                onClick={() => setGameFocus(step)}
              >
                <span className="tabular-nums text-[10px] opacity-70">{i + 1}</span>
                {pageLabel(step, g)}
                {on ? <motion.span layoutId="game-step-rail" className="absolute inset-x-1 -bottom-px h-0.5 rounded-full bg-accent" transition={rail} aria-hidden /> : null}
              </button>
            );
          })}
        </nav>
      ) : null}

      {gameId && !ready ? (
        <EmptyState className="flex-1" icon={<Gamepad2 className="size-5" />} title={g.loadingWorld} body={g.homeHint} />
      ) : gameId && loadErr && !current ? (
        <EmptyState
          className="flex-1"
          icon={<Gamepad2 className="size-5" />}
          title={g.loadFailed}
          body={loadErr}
          action={<Button size="sm" onClick={() => {
            setReady(false);
            setLoadErr("");
            void loadCurrent(gameId).then(() => setReady(true)).catch((e) => {
              setLoadErr(api.errMessage(e));
              setReady(true);
            });
          }}>{g.retryLoad}</Button>}
        />
      ) : current && page === "intent" ? (
        <GoalPage
          copy={g}
          title={title}
          intent={intent}
          camera={camera}
          busy={!!busy}
          onTitle={setTitle}
          onIntent={setIntent}
          onCamera={setCamera}
          onSave={() => { void saveIntent(); }}
          onAsk={() => { void saveIntent().then((ok) => { if (!ok) return; pendingPlay.current = true; setGameFocus("build"); ask("build"); }); }}
        />
      ) : current && page === "build" ? (
        <MakePage
          copy={g}
          row={current}
          busy={!!busy}
          onAsk={() => ask("build")}
          onMakeNow={() => void finishBuild(current)}
          onPlay={() => setGameFocus("play")}
        />
      ) : current && page === "check" ? (
        <ReviewPage copy={g} row={current} busy={!!busy} onAsk={() => ask("check")} />
      ) : current ? (
        <div className="relative flex min-h-0 flex-1 flex-col" data-testid="game-view-play">
          <div className="relative min-h-0 min-w-0 flex-1 overflow-hidden bg-media-surface" data-testid="game-viewport">
            {playable && current.preview_html ? (
              <div className="absolute inset-0">
                <SandboxedFrame html={current.preview_html} title={current.title || copy.video.newGame} fill allowScripts />
              </div>
            ) : (
              <PlayEmpty copy={g} busy={!!busy} onAsk={() => { setGameFocus("build"); ask("build"); }} />
            )}
            {playable ? (
              <p className="pointer-events-none absolute bottom-3 left-3 z-10 flex items-center gap-1.5 text-[11px] text-foreground/55">
                <Kbd>W</Kbd><Kbd>A</Kbd><Kbd>S</Kbd><Kbd>D</Kbd>
                <span>{g.playHint}</span>
              </p>
            ) : null}
          </div>
        </div>
      ) : null}

      <ConfirmDialog
        open={!!kill}
        title={g.delete}
        body={g.deleteBody}
        danger
        onCancel={() => setKill(null)}
        onConfirm={async () => {
          if (!kill) return;
          await run("delete", async () => {
            await api.video.deleteGame(kill.id);
            if (gameId === kill.id) {
              setGameFocus("");
              setGameId("");
              setCurrent(null);
            }
          });
          setKill(null);
        }}
      />
    </div>
  );
}
