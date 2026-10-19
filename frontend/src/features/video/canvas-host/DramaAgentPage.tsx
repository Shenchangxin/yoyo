import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { App, Button, Select } from "antd";
import { Film, LayoutGrid, Paperclip } from "lucide-react";
import { Tooltip } from "../../../components/ui/tooltip";
import { ModelPicker } from "@yingce/components/model-picker";
import { refreshSystemChannels } from "@yingce/lib/user-session";
import { modelOptionName, selectableModelsByCapability, useConfigStore, useEffectiveConfig } from "@yingce/stores/use-config-store";
import * as api from "../../../lib/client";
import type { Item } from "../../../lib/protocol";
import { mergeItem, subscribeSession } from "../../../lib/stream";
import { useCopy } from "../../../lib/i18n";
import { useUI } from "../../../lib/store";
import { Composer } from "../../Composer";
import { Transcript } from "../../Transcript";
import { useDramaSelection } from "../workshop-store";
import { bindVideoThread, ensureDramaEpisode, ingestSeriesFile, ingestSeriesText } from "../bind";
import { looksLikeSeries, PRESET_STYLES, shownEpisodeTitle, shownSeriesTitle, styleLabel } from "../drama-lib";
import { useCanvasHost } from "./session";
import "@yingce/pages/create/creation-product.css";
import "@yingce/pages/create/creation-scrollbars.css";

type Drama = { id: string; title: string; style?: string };
type StyleOpt = { value: string; name: string };
type Episode = { id: string; drama_id: string; title: string; episode_number?: number };

export default function DramaAgentPage() {
  const { message } = App.useApp();
  const copy = useCopy();
  const sessionId = useCanvasHost((s) => s.sessionId);
  const videoBoard = useUI((s) => s.videoBoard);
  const setVideoBoard = useUI((s) => s.setVideoBoard);
  const openSettings = useUI((s) => s.openSettings);
  const config = useEffectiveConfig();
  const updateConfig = useConfigStore((s) => s.updateConfig);
  const textModels = useMemo(() => selectableModelsByCapability(config, "text"), [config]);
  const selectedTextModel = textModels.includes(config.textModel) ? config.textModel : (textModels[0] || "");
  const dramaId = useDramaSelection((s) => s.dramaId);
  const episodeId = useDramaSelection((s) => s.episodeId);
  const dramaStyle = useDramaSelection((s) => s.dramaStyle);
  const setDramaId = useDramaSelection((s) => s.setDramaId);
  const setEpisodeId = useDramaSelection((s) => s.setEpisodeId);
  const setDramaStyle = useDramaSelection((s) => s.setDramaStyle);
  const threadScrollRef = useRef<HTMLElement>(null);
  const itemsRef = useRef<Item[]>([]);
  const [busy, setBusy] = useState(false);
  const [items, setItems] = useState<Item[]>([]);
  const [dramas, setDramas] = useState<Drama[]>([]);
  const [episodes, setEpisodes] = useState<Episode[]>([]);
  const [styles, setStyles] = useState<StyleOpt[]>(PRESET_STYLES.map((s) => ({ value: s.value, name: s.name })));

  const isEmpty = items.length === 0;
  const draftKey = `drama:${sessionId || "new"}`;

  const loadDramas = useCallback(async () => {
    const list = (await api.video.listDramas()) as Drama[];
    setDramas(Array.isArray(list) ? list : []);
  }, []);

  useEffect(() => {
    void loadDramas().catch((err) => message.error(api.errMessage(err)));
  }, [loadDramas, message]);

  useEffect(() => {
    void api.video.styles().then((list) => {
      const rows = Array.isArray(list) ? list.filter((s: StyleOpt) => s?.value) : [];
      if (rows.length) setStyles(rows.map((s: StyleOpt) => ({ value: s.value, name: s.name || s.value })));
    }).catch(() => {});
  }, []);

  useEffect(() => {
    const hit = dramas.find((d) => d.id === dramaId);
    if (hit?.style) setDramaStyle(hit.style);
  }, [dramaId, dramas, setDramaStyle]);

  useEffect(() => {
    void refreshSystemChannels().catch(() => {});
  }, []);

  useEffect(() => {
    if (!dramaId) {
      setEpisodes([]);
      return;
    }
    let cancelled = false;
    void api.video.episodes(dramaId).then((list) => {
      if (cancelled) return;
      const eps = Array.isArray(list) ? list : [];
      setEpisodes(eps);
      if (!useDramaSelection.getState().episodeId && eps[0]) setEpisodeId(eps[0].id);
    }).catch(() => {
      if (!cancelled) setEpisodes([]);
    });
    return () => {
      cancelled = true;
    };
  }, [dramaId, setEpisodeId]);

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

  useEffect(() => {
    const el = threadScrollRef.current;
    if (!el) return;
    el.scrollTop = el.scrollHeight;
  }, [items, busy]);

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

  async function pickStyle(value: string) {
    setDramaStyle(value);
    if (!dramaId) return;
    try {
      await api.video.updateDrama({ id: dramaId, style: value });
      await loadDramas();
    } catch (err) {
      message.error(api.errMessage(err));
    }
  }

  async function send(text = "") {
    const next = text.trim();
    if (!next) return;
    if (busy) {
      message.info(copy.video.agentSending);
      return;
    }
    const session = await sid();
    if (!session) {
      message.error(copy.video.failed);
      return;
    }
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
        setBusy(false);
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
      message.error(api.errMessage(err));
    } finally {
      setBusy(false);
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
      message.error(api.errMessage(err));
    } finally {
      setBusy(false);
    }
  }

  const dramaLeading = (
    <>
      <Select
        size="small"
        variant="borderless"
        popupMatchSelectWidth={false}
        getPopupContainer={() => document.body}
        className="drama-agent-select"
        value={dramaStyle || "3d"}
        placeholder={copy.video.style}
        aria-label={copy.video.style}
        onChange={(value) => void pickStyle(value)}
        options={styles.map((row) => ({ value: row.value, label: styleLabel(copy.video.styleNames, row.value, row.name) }))}
      />
      {!isEmpty && !videoBoard ? (
        <>
          <Select
            size="small"
            variant="borderless"
            popupMatchSelectWidth={false}
            getPopupContainer={() => document.body}
            className="drama-agent-select"
            value={dramaId || undefined}
            placeholder={copy.video.pickSeries}
            aria-label={copy.video.pickSeries}
            onChange={(id) => {
              setDramaId(id);
              setEpisodeId("");
              const hit = dramas.find((d) => d.id === id);
              if (hit?.style) setDramaStyle(hit.style);
            }}
            options={dramas.map((d) => ({ value: d.id, label: shownSeriesTitle(d.title, copy.video.untitled) }))}
          />
          <Select
            size="small"
            variant="borderless"
            popupMatchSelectWidth={false}
            getPopupContainer={() => document.body}
            className="drama-agent-select"
            value={episodeId || undefined}
            placeholder={copy.video.pickEpisode}
            aria-label={copy.video.pickEpisode}
            disabled={!dramaId}
            onChange={setEpisodeId}
            options={episodes.map((e) => ({ value: e.id, label: shownEpisodeTitle(e.title, e.episode_number, copy.video.episodeN) }))}
          />
        </>
      ) : null}
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
        className="creation-model-picker drama-model-picker"
        placeholder={copy.video.textModel}
        showSelectedPrice={false}
        showOptionPrices={false}
        onMissingConfig={() => openSettings("provider")}
      />
      <Tooltip content={copy.video.ingestFileHint}>
        <button type="button" className="sc-chip-btn" data-testid="drama-import-novel" aria-label={copy.video.mapFile} disabled={busy} onClick={() => void importNovel()}>
          <Paperclip className="size-3.5" />
          <span>{copy.video.mapFile}</span>
        </button>
      </Tooltip>
    </>
  );

  const composer = (
    <Composer
      draftKey={draftKey}
      running={busy}
      disabled={false}
      onSend={(opts) => { void send(opts?.text || ""); }}
      onStop={() => {
        setBusy(false);
        if (sessionId) void api.interrupt(sessionId).catch(() => {});
      }}
      model={selectedTextModel ? modelOptionName(selectedTextModel) : undefined}
      flush
      compact
      hero={isEmpty && !videoBoard}
      perch={isEmpty && !videoBoard}
      placeholder={copy.video.agentPlaceholder}
      inputTestId="drama-agent-input"
      toolbarLeading={dramaLeading}
    />
  );

  const path = [
    { n: "1", title: copy.video.path1, hint: copy.video.path1Hint },
    { n: "2", title: copy.video.path2, hint: copy.video.path2Hint },
    { n: "3", title: copy.video.path3, hint: copy.video.path3Hint },
  ];

  return (
    <div
      className="creation-home drama-agent-page relative flex h-full min-h-0 flex-col overflow-hidden"
      data-testid="drama-agent-page"
      data-empty={isEmpty ? "true" : "false"}
      data-board={videoBoard ? "true" : "false"}
    >
      {isEmpty && videoBoard ? (
        <div className="drama-board-chat">
          <div className="drama-board-chat-spacer">
            <p className="drama-board-chat-hint">{copy.video.agentHint}</p>
          </div>
          <div className="creation-empty-composer">{composer}</div>
        </div>
      ) : isEmpty ? (
        <main ref={threadScrollRef} className="creation-empty-workspace creation-scrollbar">
          <div className="creation-home-heading">
            <h1 data-testid="drama-agent-heading">{copy.video.agentTitle}</h1>
            <p data-testid="drama-agent-hint">{copy.video.agentHint}</p>
          </div>
          <section className="creation-launchpad" aria-label={copy.video.desk}>
            <div className="creation-composer-stage is-home-mode">
              <div className="creation-empty-composer">{composer}</div>
            </div>
            <ol className="drama-path" data-testid="drama-path" aria-label={copy.video.pathLabel}>
              {path.map((step) => (
                <li key={step.n} className="drama-path-step">
                  <span className="drama-path-n" aria-hidden>{step.n}</span>
                  <span className="drama-path-copy">
                    <strong>{step.title}</strong>
                    <em>{step.hint}</em>
                  </span>
                </li>
              ))}
            </ol>
            {dramas.length > 0 ? (
              <div className="drama-existing" data-testid="drama-existing">
                <p>{copy.video.continueSeries}</p>
                <div className="drama-existing-list">
                  {dramas.slice(0, 6).map((d) => (
                    <button
                      key={d.id}
                      type="button"
                      className="drama-existing-chip"
                      onClick={() => {
                        setDramaId(d.id);
                        setEpisodeId("");
                        if (d.style) setDramaStyle(d.style);
                        setVideoBoard(true);
                      }}
                    >
                      {shownSeriesTitle(d.title, copy.video.untitled)}
                    </button>
                  ))}
                </div>
              </div>
            ) : null}
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
        <div className="creation-thread-workbench min-h-0 flex-1">
          <header className="creation-thread-toolbar">
            <div className="creation-toolbar-shots">
              <span className="drama-thread-label"><Film aria-hidden />{copy.video.drama}</span>
            </div>
            <div className="creation-toolbar-actions">
              <Button size="small" icon={<LayoutGrid className="size-3.5" />} onClick={() => setVideoBoard(true)} data-testid="drama-open-board">{copy.video.openBoard}</Button>
            </div>
          </header>
          <main ref={threadScrollRef} className="creation-thread-scroll creation-scrollbar">
            <Transcript
              items={items}
              approvals={[]}
              running={busy}
              compact
              flush
              onResolve={() => {}}
              onPrompt={(text) => useUI.getState().setDraft(draftKey, text)}
              onRetry={() => {
                const last = [...items].reverse().find((it) => it.type === "user");
                if (last?.text) useUI.getState().setDraft(draftKey, last.text);
              }}
            />
          </main>
          <section className="creation-thread-composer">{composer}</section>
        </div>
      )}
    </div>
  );
}
