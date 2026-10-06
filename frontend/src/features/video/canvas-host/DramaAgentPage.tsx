import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { App, Button, Select } from "antd";
import { ArrowUp, Film, LayoutGrid, Paperclip } from "lucide-react";
import { Tooltip } from "@yingce/components/ui/base/tooltip";
import { ModelPicker } from "@yingce/components/model-picker";
import { refreshSystemChannels } from "@yingce/lib/user-session";
import { modelOptionName, selectableModelsByCapability, useConfigStore, useEffectiveConfig } from "@yingce/stores/use-config-store";
import { CreationMessageView } from "@yingce/pages/create/creation-workspace";
import type { CreationMessage } from "@yingce/pages/create/creation-types";
import { cn } from "@yingce/lib/utils";
import * as api from "../../../lib/client";
import type { Item } from "../../../lib/protocol";
import { mergeItem, subscribeSession } from "../../../lib/stream";
import { useCopy } from "../../../lib/i18n";
import { useUI } from "../../../lib/store";
import { useDramaSelection } from "../workshop-store";
import { bindVideoThread, ensureDramaEpisode, ingestSeriesFile, ingestSeriesText } from "../bind";
import { looksLikeSeries, shownEpisodeTitle, shownSeriesTitle } from "../drama-lib";
import { useCanvasHost } from "./session";
import "@yingce/pages/create/creation-product.css";
import "@yingce/pages/create/creation-scrollbars.css";

type Drama = { id: string; title: string };
type Episode = { id: string; drama_id: string; title: string; episode_number?: number };

function visibleItems(items: Item[]): CreationMessage[] {
  const out: CreationMessage[] = [];
  for (const item of items) {
    if (item.type === "user" && item.source !== "steer") {
      out.push({ id: item.key, role: "user", mode: "text", content: item.text, createdAt: item.ts || new Date().toISOString(), status: "done" });
    } else if (item.type === "assistant") {
      out.push({
        id: item.key,
        role: "assistant",
        mode: "text",
        content: item.text,
        createdAt: item.ts || new Date().toISOString(),
        status: item.delta ? "streaming" : "done",
        reasoning: typeof item.payload?.reasoning === "string" ? item.payload.reasoning : undefined,
      });
    } else if (item.type === "error") {
      out.push({ id: item.key, role: "assistant", mode: "text", content: "", createdAt: item.ts || new Date().toISOString(), status: "error", error: item.text || "失败" });
    }
  }
  return out;
}

export default function DramaAgentPage() {
  const { message } = App.useApp();
  const copy = useCopy();
  const sessionId = useCanvasHost((s) => s.sessionId);
  const setVideoBoard = useUI((s) => s.setVideoBoard);
  const openSettings = useUI((s) => s.openSettings);
  const config = useEffectiveConfig();
  const updateConfig = useConfigStore((s) => s.updateConfig);
  const textModels = useMemo(() => selectableModelsByCapability(config, "text"), [config]);
  const selectedTextModel = textModels.includes(config.textModel) ? config.textModel : (textModels[0] || "");
  const dramaId = useDramaSelection((s) => s.dramaId);
  const episodeId = useDramaSelection((s) => s.episodeId);
  const setDramaId = useDramaSelection((s) => s.setDramaId);
  const setEpisodeId = useDramaSelection((s) => s.setEpisodeId);
  const composerFocusRef = useRef<HTMLTextAreaElement>(null);
  const threadScrollRef = useRef<HTMLElement>(null);
  const itemsRef = useRef<Item[]>([]);
  const [prompt, setPrompt] = useState("");
  const [busy, setBusy] = useState(false);
  const [items, setItems] = useState<Item[]>([]);
  const [dramas, setDramas] = useState<Drama[]>([]);
  const [episodes, setEpisodes] = useState<Episode[]>([]);

  const messages = useMemo(() => visibleItems(items), [items]);
  const isEmpty = messages.length === 0;
  const canSubmit = Boolean(prompt.trim()) && !busy;

  const loadDramas = useCallback(async () => {
    const list = (await api.video.listDramas()) as Drama[];
    setDramas(Array.isArray(list) ? list : []);
  }, []);

  useEffect(() => {
    void loadDramas().catch(() => {});
  }, [loadDramas]);

  useEffect(() => {
    void refreshSystemChannels().catch(() => {});
  }, []);

  useEffect(() => {
    if (!dramaId) {
      setEpisodes([]);
      return;
    }
    void api.video.episodes(dramaId).then((list) => {
      const eps = Array.isArray(list) ? list : [];
      setEpisodes(eps);
      if (!episodeId && eps[0]) setEpisodeId(eps[0].id);
    }).catch(() => setEpisodes([]));
  }, [dramaId, episodeId, setEpisodeId]);

  useEffect(() => {
    if (!sessionId) return;
    itemsRef.current = [];
    setItems([]);
    return subscribeSession(
      sessionId,
      (item) => {
        itemsRef.current = mergeItem(itemsRef.current, item);
        setItems(itemsRef.current.slice());
        if (item.type === "turn_end" || item.type === "error") setBusy(false);
      },
      (seed) => {
        itemsRef.current = seed;
        setItems(seed.slice());
      },
    );
  }, [sessionId]);

  async function sid() {
    const existing = sessionId || (typeof window !== "undefined" ? String((window as Window & { __YOYO_VIDEO_SESSION__?: string }).__YOYO_VIDEO_SESSION__ || "") : "");
    if (existing) return existing;
    return useCanvasHost.getState().ensureSession();
  }

  async function ensureCurrent(content = "") {
    const id = await ensureDramaEpisode({
      title: "",
      episodeTitle: "",
      content,
      ratio: "9:16",
    });
    await loadDramas();
    return id;
  }

  async function send(text = prompt) {
    const next = text.trim();
    if (!next || busy) return;
    const session = await sid();
    if (!session) {
      message.error(copy.video.failed);
      return;
    }
    setPrompt("");
    setBusy(true);
    try {
      if (looksLikeSeries(next)) {
        const drama = await ingestSeriesText(next);
        await api.video.bind(session, "", drama);
        if (selectedTextModel) {
          await api.setSessionModel(session, modelOptionName(selectedTextModel)).catch(() => {});
        }
        await api.video.stage(session, "", "outline", drama);
        await loadDramas();
        setVideoBoard(true);
        return;
      }
      await ensureCurrent(next);
      await bindVideoThread(session).catch(() => {});
      if (selectedTextModel) {
        await api.setSessionModel(session, modelOptionName(selectedTextModel)).catch(() => {});
      }
      await api.send(session, next);
      setVideoBoard(true);
    } catch (err) {
      setBusy(false);
      message.error(api.errMessage(err));
    }
  }

  async function importNovel() {
    if (busy) return;
    const paths = await api.pickFiles();
    const path = (paths || []).find((p) => /\.(txt|md|markdown|docx|pdf)$/i.test(p)) || (paths || [])[0];
    if (!path) return;
    const session = await sid();
    if (!session) {
      message.error(copy.video.failed);
      return;
    }
    setBusy(true);
    try {
      const drama = await ingestSeriesFile(path);
      await api.video.bind(session, "", drama);
      if (selectedTextModel) {
        await api.setSessionModel(session, modelOptionName(selectedTextModel)).catch(() => {});
      }
      await api.video.stage(session, "", "outline", drama);
      await loadDramas();
      setVideoBoard(true);
    } catch (err) {
      setBusy(false);
      message.error(api.errMessage(err));
    }
  }

  const composer = (
    <div className="creation-composer-shell">
      <div className={`creation-chat-composer is-${isEmpty ? "empty" : "thread"}`}>
        <div className="creation-chat-writing-surface">
          <div className="creation-chat-editor">
            <textarea
              ref={composerFocusRef}
              className="creation-chat-mention-editor creation-scrollbar"
              style={{ color: "var(--creation-text)" }}
              value={prompt}
              onChange={(event) => setPrompt(event.target.value)}
              onKeyDown={(event) => {
                if (event.key === "Enter" && !event.shiftKey) {
                  event.preventDefault();
                  void send();
                }
              }}
              placeholder={copy.video.agentPlaceholder}
              aria-label={copy.video.drama}
              spellCheck={false}
              disabled={busy}
              data-testid="drama-agent-input"
            />
          </div>
        </div>
        <footer className="creation-chat-dock">
          <div className="creation-chat-controls">
            {!isEmpty ? (
              <>
                <Select
                  size="small"
                  variant="borderless"
                  className="drama-agent-select min-w-28"
                  value={dramaId || undefined}
                  placeholder={copy.video.pickSeries}
                  aria-label={copy.video.pickSeries}
                  onChange={(id) => { setDramaId(id); setEpisodeId(""); }}
                  options={dramas.map((d) => ({ value: d.id, label: shownSeriesTitle(d.title, copy.video.untitled) }))}
                />
                <Select
                  size="small"
                  variant="borderless"
                  className="drama-agent-select min-w-24"
                  value={episodeId || undefined}
                  placeholder={copy.video.pickEpisode}
                  aria-label={copy.video.pickEpisode}
                  disabled={!dramaId}
                  onChange={setEpisodeId}
                  options={episodes.map((e) => ({ value: e.id, label: shownEpisodeTitle(e.title, e.episode_number, copy.video.episodeN) }))}
                />
                <ModelPicker
                  config={config}
                  value={selectedTextModel}
                  onChange={(model) => {
                    updateConfig("textModel", model);
                    const session = sessionId || (typeof window !== "undefined" ? String((window as Window & { __YOYO_VIDEO_SESSION__?: string }).__YOYO_VIDEO_SESSION__ || "") : "");
                    if (session) void api.setSessionModel(session, modelOptionName(model)).catch(() => {});
                  }}
                  capability="text"
                  variant="creation"
                  className="creation-model-picker"
                  placeholder={copy.video.textModel}
                  showSelectedPrice={false}
                  showOptionPrices={false}
                  onMissingConfig={() => openSettings("provider")}
                />
                <Tooltip title={copy.video.openBoard}>
                  <button type="button" className="creation-chat-control" data-testid="drama-open-board" aria-label={copy.video.openBoard} onClick={() => setVideoBoard(true)}>
                    <LayoutGrid /><span>{copy.video.openBoard}</span>
                  </button>
                </Tooltip>
              </>
            ) : !selectedTextModel ? (
              <ModelPicker
                config={config}
                value={selectedTextModel}
                onChange={(model) => {
                  updateConfig("textModel", model);
                  const session = sessionId || (typeof window !== "undefined" ? String((window as Window & { __YOYO_VIDEO_SESSION__?: string }).__YOYO_VIDEO_SESSION__ || "") : "");
                  if (session) void api.setSessionModel(session, modelOptionName(model)).catch(() => {});
                }}
                capability="text"
                variant="creation"
                className="creation-model-picker"
                placeholder={copy.video.textModel}
                showSelectedPrice={false}
                showOptionPrices={false}
                onMissingConfig={() => openSettings("provider")}
              />
            ) : null}
            <Tooltip title={copy.video.ingestFileHint}>
              <button type="button" className="creation-chat-control" data-testid="drama-import-novel" aria-label={copy.video.mapFile} disabled={busy} onClick={() => void importNovel()}>
                <Paperclip /><span>{copy.video.mapFile}</span>
              </button>
            </Tooltip>
          </div>
          <Button
            type="text"
            className="creation-submit"
            disabled={!canSubmit}
            onClick={() => void send()}
            aria-label={busy ? copy.video.agentSending : copy.video.agentSend}
          >
            <span className="creation-submit-action" aria-hidden>
              <ArrowUp className="size-4" />
              <span>{busy ? copy.video.agentSending : copy.video.agentSend}</span>
            </span>
          </Button>
        </footer>
      </div>
    </div>
  );

  const path = [
    { n: "1", title: copy.video.path1, hint: copy.video.path1Hint },
    { n: "2", title: copy.video.path2, hint: copy.video.path2Hint },
    { n: "3", title: copy.video.path3, hint: copy.video.path3Hint },
  ];

  return (
    <div className="creation-home relative flex h-full min-h-0 flex-col overflow-hidden" data-testid="drama-agent-page">
      {isEmpty ? (
        <main ref={threadScrollRef} className="creation-empty-workspace creation-scrollbar">
          <div className="creation-home-heading">
            <h1 data-testid="drama-agent-heading">{copy.video.agentTitle}</h1>
            <p data-testid="drama-agent-hint">{copy.video.agentHint}</p>
          </div>
          <section className="creation-launchpad" aria-label={copy.video.desk}>
            <ol className="drama-path" data-testid="drama-path" aria-label={copy.video.pathLabel}>
              {path.map((step) => (
                <li key={step.n} className="drama-path-step">
                  <span className="drama-path-n" aria-hidden>{step.n}</span>
                  <span className="drama-path-copy">
                    <strong>{step.title}</strong>
                    <span>{step.hint}</span>
                  </span>
                </li>
              ))}
            </ol>
            <div className="creation-composer-stage is-home-mode">
              <div className="creation-empty-composer">{composer}</div>
            </div>
            <button
              type="button"
              className="drama-open-existing"
              data-testid="drama-open-board"
              onClick={() => setVideoBoard(true)}
            >
              {copy.video.openBoardExisting}
            </button>
          </section>
        </main>
      ) : (
        <div className="creation-thread-workbench">
          <header className="creation-thread-toolbar">
            <div className="creation-toolbar-shots">
              <span className="creation-rail-trigger"><Film />{copy.video.drama}</span>
            </div>
            <div className="creation-toolbar-actions">
              <Button size="small" icon={<LayoutGrid className="size-3.5" />} onClick={() => setVideoBoard(true)} data-testid="drama-open-board">{copy.video.openBoard}</Button>
            </div>
          </header>
          <main ref={threadScrollRef} className="creation-thread-scroll creation-scrollbar">
            <section className="creation-thread-stage">
              <div className="creation-results">
                {messages.map((item) => (
                  <div key={item.id} className="creation-thread-message">
                    <CreationMessageView
                      item={item}
                      shotNumber={0}
                      onRetryFailure={() => { setPrompt(item.content); window.requestAnimationFrame(() => composerFocusRef.current?.focus()); }}
                      onCreateVariant={() => {}}
                      onContinueCanvas={() => {}}
                      openingCanvas={false}
                      onEditUserMessage={(text) => { setPrompt(text); window.requestAnimationFrame(() => composerFocusRef.current?.focus()); }}
                    />
                  </div>
                ))}
              </div>
            </section>
          </main>
          <section className={cn("creation-thread-composer")}>{composer}</section>
        </div>
      )}
    </div>
  );
}
