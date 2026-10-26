/**
 * 本文件拥有短剧代理页：风格、系列、输入和线程。不拥有短剧看板的工序。
 * 发送、导入、绑定和开阶段会改服务端。像系列的正文走导入，其他正文走当前会话发送。
 */
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
import { DramaStylePicker } from "../DramaStylePicker";
import { looksLikeSeries, mergeDramaStyles, PRESET_STYLES, shownEpisodeTitle, shownSeriesTitle } from "../drama-lib";
import { canvasOverlayHost } from "./design-system/popup-container";
import { useCanvasHost } from "./session";
import "@yingce/pages/create/creation-product.css";
import "@yingce/pages/create/creation-scrollbars.css";

type Drama = { id: string; title: string; style?: string };
type StyleOpt = { value: string; name: string };
type Episode = { id: string; drama_id: string; title: string; episode_number?: number };

/** DramaAgentPage 在空线程时显示入口，有消息后显示记录。系列正文会打开看板。 */
export default function DramaAgentPage() {
  const { message } = App.useApp();
  const copy = useCopy();
  const sessionId = useCanvasHost(/** s 取出 sessionId。 */ (s) => s.sessionId);
  const videoBoard = useUI(/** s 取出 videoBoard。 */ (s) => s.videoBoard);
  const setVideoBoard = useUI(/** s 取出 setVideoBoard。 */ (s) => s.setVideoBoard);
  const openSettings = useUI(/** s 取出 openSettings。 */ (s) => s.openSettings);
  const config = useEffectiveConfig();
  const updateConfig = useConfigStore(/** s 取出 updateConfig。 */ (s) => s.updateConfig);
  const textModels = useMemo(() => selectableModelsByCapability(config, "text"), [config]);
  const selectedTextModel = textModels.includes(config.textModel) ? config.textModel : (textModels[0] || "");
  const dramaId = useDramaSelection(/** s 取出 dramaId。 */ (s) => s.dramaId);
  const episodeId = useDramaSelection(/** s 取出 episodeId。 */ (s) => s.episodeId);
  const dramaStyle = useDramaSelection(/** s 取出 dramaStyle。 */ (s) => s.dramaStyle);
  const setDramaId = useDramaSelection(/** s 取出 setDramaId。 */ (s) => s.setDramaId);
  const setEpisodeId = useDramaSelection(/** s 取出 setEpisodeId。 */ (s) => s.setEpisodeId);
  const setDramaStyle = useDramaSelection(/** s 取出 setDramaStyle。 */ (s) => s.setDramaStyle);
  const threadScrollRef = useRef<HTMLElement>(null);
  const itemsRef = useRef<Item[]>([]);
  const [busy, setBusy] = useState(false);
  const [items, setItems] = useState<Item[]>([]);
  const [threadOpen, setThreadOpen] = useState(false);
  const [dramas, setDramas] = useState<Drama[]>([]);
  const [episodes, setEpisodes] = useState<Episode[]>([]);
  const [styles, setStyles] = useState<StyleOpt[]>(PRESET_STYLES.map(/** s 留下 value 和 name。 */ (s) => ({ value: s.value, name: s.name })));

  const isEmpty = !threadOpen;
  const draftKey = "drama:home";

  /** 读取系列列表。 */
  const loadDramas = useCallback(async () => {
    const list = (await api.video.listDramas()) as Drama[];
    setDramas(Array.isArray(list) ? list : []);
  }, []);

  useEffect(() => {
    void loadDramas().catch(/** err 交给 errMessage 后弹出。 */ (err) => message.error(api.errMessage(err)));
  }, [loadDramas, message]);

  useEffect(() => {
    void api.video.styles().then(/** list 是数组才交给 mergeDramaStyles，否则交给空数组。 */ (list) => {
      setStyles(mergeDramaStyles(Array.isArray(list) ? list : []));
    }).catch(() => {});
  }, []);

  useEffect(() => {
    const hit = dramas.find(/** d 用 id 和 dramaId 比对。 */ (d) => d.id === dramaId);
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
    void api.video.episodes(dramaId).then(/** list 是数组才当成集；还没有选集时记下第一条的 id。 */ (list) => {
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
      /** item 合并进记录；类型是 turn_end 或 error 时清掉忙碌。 */
      (item) => {
        itemsRef.current = mergeItem(itemsRef.current, item);
        setItems(itemsRef.current.slice());
        if (item.type === "turn_end" || item.type === "error") setBusy(false);
      },
      /** seed 换成当前记录，并刷新一份副本。 */
      (seed) => {
        itemsRef.current = seed;
        setItems(seed.slice());
      },
    );
  }, [sessionId]);

  useEffect(() => {
    if (isEmpty) return;
    const el = threadScrollRef.current;
    if (!el) return;
    el.scrollTop = el.scrollHeight;
  }, [items, busy, isEmpty]);

  /** 取出已有会话，没有就向画布宿主要。 */
  async function sid() {
    const existing = sessionId || (typeof window !== "undefined" ? String((window as Window & { __YOYO_VIDEO_SESSION__?: string }).__YOYO_VIDEO_SESSION__ || "") : "");
    if (existing) return existing;
    return useCanvasHost.getState().ensureSession();
  }

  /** 保证有一集，比例是 9:16，并刷新系列列表。 content 作为新集正文交给 ensureDramaEpisode。 */
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

  /** 记下风格；已有系列就写回。 value 记成风格，有系列时作为 style 写回。 */
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

  /** 像系列的正文走导入和大纲阶段，否则写入当前集并发送到会话。 text 去掉空白，空着就返回；像系列就拿这段导入，否则写入当前集并发送。 */
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
    setThreadOpen(true);
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

  /** 选一份文本文件导入系列，并打开大纲阶段。 */
  async function importNovel() {
    if (busy) return;
    const paths = await api.pickFiles();
    const path = (paths || []).find(/** p 用后缀判断是不是文本文件。 */ (p) => /\.(txt|md|markdown|docx|pdf)$/i.test(p)) || (paths || [])[0];
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
      <DramaStylePicker
        value={dramaStyle || "3d"}
        styles={styles}
        names={copy.video.styleNames}
        label={copy.video.style}
        title={copy.video.stylePickerTitle}
        search={copy.video.styleSearch}
        empty={copy.video.styleEmpty}
        groups={copy.video.styleGroups}
        onChange={/** value 交给 pickStyle。 */ (value) => void pickStyle(value)}
      />
      {!isEmpty && !videoBoard ? (
        <>
          <Select
            size="small"
            variant="borderless"
            popupMatchSelectWidth={false}
            placement="topLeft"
            getPopupContainer={canvasOverlayHost}
            className="drama-agent-select"
            popupClassName="drama-agent-select-dropdown"
            value={dramaId || undefined}
            placeholder={copy.video.pickSeries}
            aria-label={copy.video.pickSeries}
            onChange={/** id 写成当前系列并清掉集；找到系列且有 style 就记下。 */ (id) => {
              setDramaId(id);
              setEpisodeId("");
              const hit = dramas.find(/** d 用 id 和刚选的系列比对。 */ (d) => d.id === id);
              if (hit?.style) setDramaStyle(hit.style);
            }}
            options={dramas.map(/** d 用 id 当值，用标题当标签。 */ (d) => ({ value: d.id, label: shownSeriesTitle(d.title, copy.video.untitled) }))}
          />
          <Select
            size="small"
            variant="borderless"
            popupMatchSelectWidth={false}
            placement="topLeft"
            getPopupContainer={canvasOverlayHost}
            className="drama-agent-select"
            popupClassName="drama-agent-select-dropdown"
            value={episodeId || undefined}
            placeholder={copy.video.pickEpisode}
            aria-label={copy.video.pickEpisode}
            disabled={!dramaId}
            onChange={setEpisodeId}
            options={episodes.map(/** e 用 id 当值，用集标题当标签。 */ (e) => ({ value: e.id, label: shownEpisodeTitle(e.title, e.episode_number, copy.video.episodeN) }))}
          />
        </>
      ) : null}
      <Tooltip content={copy.video.ingestFileHint}>
        <button type="button" className="sc-chip-btn" data-testid="drama-import-novel" aria-label={copy.video.mapFile} disabled={busy} onClick={() => void importNovel()}>
          <Paperclip className="size-3.5" />
          <span>{copy.video.mapFile}</span>
        </button>
      </Tooltip>
    </>
  );

  const dramaModel = (
    <ModelPicker
      config={config}
      value={selectedTextModel}
      onChange={/** model 写进 textModel，有会话时换成选项名交给 setSessionModel。 */ (model) => {
        updateConfig("textModel", model);
        const session = sessionId || (typeof window !== "undefined" ? String((window as Window & { __YOYO_VIDEO_SESSION__?: string }).__YOYO_VIDEO_SESSION__ || "") : "");
        if (session) void api.setSessionModel(session, modelOptionName(model)).catch(() => {});
      }}
      capability="text"
      variant="creation"
      compact
      placement="topLeft"
      getPopupContainer={canvasOverlayHost}
      className="creation-model-picker sc-chip-btn"
      popoverClassName="creation-model-picker-popover"
      testId="drama-model-picker"
      placeholder={copy.video.textModel}
      showSelectedPrice={false}
      showOptionPrices
      onMissingConfig={() => openSettings("provider")}
    />
  );

  const composer = (
    <Composer
      draftKey={draftKey}
      running={busy}
      disabled={false}
      onSend={/** opts 有 text 就交给 send，否则交给空字符串。 */ (opts) => { void send(opts?.text || ""); }}
      onStop={() => {
        setBusy(false);
        if (sessionId) void api.interrupt(sessionId).catch(() => {});
      }}
      flush
      compact
      hero={isEmpty && !videoBoard}
      perch={isEmpty && !videoBoard}
      perchSize={88}
      placeholder={copy.video.agentPlaceholder}
      inputTestId="drama-agent-input"
      toolbarLeading={dramaLeading}
      toolbarTrailing={dramaModel}
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
              {path.map(/** step 用 n 当序号和 key，并显示 title 和 hint。 */ (step) => (
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
                  {dramas.slice(0, 6).map(/** d 用 id 选中系列并清空集，有 style 就记成风格，并显示标题。 */ (d) => (
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
              onPrompt={/** text 写进这个草稿键。 */ (text) => useUI.getState().setDraft(draftKey, text)}
              onRetry={() => {
                const last = [...items].reverse().find(/** it 看 type 是不是 user。 */ (it) => it.type === "user");
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
