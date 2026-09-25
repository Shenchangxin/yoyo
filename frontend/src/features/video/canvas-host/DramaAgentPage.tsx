import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { App, Button, Select } from "antd";
import { LayoutGroup, motion, useReducedMotion } from "motion/react";
import { ArrowUp, Clapperboard, FileText, Film, LayoutGrid, Plus, Users } from "lucide-react";
import { Tooltip } from "@yingce/components/ui/base/tooltip";
import { aceternityMotion } from "@yingce/lib/aceternity-motion";
import { useAppearanceStore } from "@yingce/stores/use-appearance-store";
import { CreationMessageView } from "@yingce/pages/create/creation-workspace";
import type { CreationMessage } from "@yingce/pages/create/creation-types";
import { cn } from "@yingce/lib/utils";
import * as api from "../../../lib/client";
import type { Item } from "../../../lib/protocol";
import { mergeItem, subscribeSession } from "../../../lib/stream";
import { useUI } from "../../../lib/store";
import { useDramaSelection } from "../workshop-store";
import { bindVideoThread } from "../bind";
import { useCanvasHost } from "./session";
import "@yingce/pages/create/creation-product.css";
import "@yingce/pages/create/creation-scrollbars.css";

type Drama = { id: string; title: string };
type Episode = { id: string; drama_id: string; title: string };

const starters = [
  { icon: FileText, title: "改写这一集", hint: "原文改成短剧剧本", prompt: "把当前集的原文改写成短剧剧本。" },
  { icon: Users, title: "提取资产", hint: "角色、场景、道具", prompt: "从剧本提取角色、场景和道具。" },
  { icon: Clapperboard, title: "拆这一章", hint: "每镜 8 到 15 秒", prompt: "把剧本拆成分镜，每镜 8 到 15 秒。" },
];

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
  const brandName = useAppearanceStore((state) => state.appearance.brandName);
  const sessionId = useCanvasHost((s) => s.sessionId);
  const setVideoBoard = useUI((s) => s.setVideoBoard);
  const dramaId = useDramaSelection((s) => s.dramaId);
  const episodeId = useDramaSelection((s) => s.episodeId);
  const setDramaId = useDramaSelection((s) => s.setDramaId);
  const setEpisodeId = useDramaSelection((s) => s.setEpisodeId);
  const reducedMotion = useReducedMotion();
  const composerFocusRef = useRef<HTMLTextAreaElement>(null);
  const threadScrollRef = useRef<HTMLElement>(null);
  const itemsRef = useRef<Item[]>([]);
  const [prompt, setPrompt] = useState("");
  const [busy, setBusy] = useState(false);
  const [items, setItems] = useState<Item[]>([]);
  const [dramas, setDramas] = useState<Drama[]>([]);
  const [episodes, setEpisodes] = useState<Episode[]>([]);
  const [hovered, setHovered] = useState<string | null>(null);

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
    if (!dramaId) {
      setEpisodes([]);
      return;
    }
    void api.video.episodes(dramaId).then((list) => {
      setEpisodes(Array.isArray(list) ? list : []);
    }).catch(() => setEpisodes([]));
  }, [dramaId]);

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

  async function send(text = prompt) {
    const next = text.trim();
    if (!next || busy) return;
    let sid = sessionId || (typeof window !== "undefined" ? String((window as Window & { __YOYO_VIDEO_SESSION__?: string }).__YOYO_VIDEO_SESSION__ || "") : "");
    if (!sid) {
      message.warning("对话还在准备，请稍后再发");
      return;
    }
    setPrompt("");
    setBusy(true);
    try {
      await bindVideoThread(sid).catch(() => {});
      await api.send(sid, next);
    } catch (err) {
      setBusy(false);
      message.error(api.errMessage(err));
    }
  }

  async function createDrama() {
    try {
      const created = await api.video.createDrama({ title: "未命名短剧", style: "3d", aspect_ratio: "9:16" });
      setDramaId(created.id);
      const ep = await api.video.createEpisode(created.id, "第一集", "");
      setEpisodeId(ep.id);
      if (sessionId) void bindVideoThread(sessionId);
      await loadDramas();
    } catch (err) {
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
              placeholder="贴一章原文，或问这一集…"
              aria-label="短剧 Agent 对话"
              spellCheck={false}
              disabled={busy}
              data-testid="drama-agent-input"
            />
          </div>
        </div>
        <footer className="creation-chat-dock">
          <div className="creation-chat-controls">
            <Select
              size="small"
              variant="borderless"
              className="drama-agent-select min-w-28"
              value={dramaId || undefined}
              placeholder="选择剧集"
              aria-label="选择剧集"
              onChange={(id) => { setDramaId(id); setEpisodeId(""); }}
              options={dramas.map((d) => ({ value: d.id, label: d.title || "未命名短剧" }))}
            />
            <Select
              size="small"
              variant="borderless"
              className="drama-agent-select min-w-24"
              value={episodeId || undefined}
              placeholder="选择集"
              aria-label="选择集"
              disabled={!dramaId}
              onChange={setEpisodeId}
              options={episodes.map((e) => ({ value: e.id, label: e.title || "未命名集" }))}
            />
            <Tooltip title="新建剧集">
              <button type="button" className="creation-chat-control" aria-label="新建剧集" onClick={() => void createDrama()}>
                <Plus /><span>新建</span>
              </button>
            </Tooltip>
            <Tooltip title="打开短剧工坊">
              <button type="button" className="creation-chat-control" data-testid="drama-open-board" aria-label="打开短剧工坊" onClick={() => setVideoBoard(true)}>
                <LayoutGrid /><span>工坊</span>
              </button>
            </Tooltip>
          </div>
          <Button
            type="text"
            className="creation-submit"
            disabled={!canSubmit}
            onClick={() => void send()}
            aria-label={busy ? "生成中" : "发送"}
          >
            <span className="creation-submit-action" aria-hidden>
              <ArrowUp className="size-4" />
              <span>{busy ? "生成中" : "开始创作"}</span>
            </span>
          </Button>
        </footer>
      </div>
    </div>
  );

  return (
    <div className="creation-home relative flex h-full min-h-0 flex-col overflow-hidden" data-testid="drama-agent-page">
      {isEmpty ? (
        <main ref={threadScrollRef} className="creation-empty-workspace creation-scrollbar">
          <div className="creation-home-heading">
            <h1>和{brandName}聊聊这部短剧</h1>
            <p>先贴一章原文，或从改写、提取、拆镜开始。</p>
          </div>
          <section className="creation-launchpad" aria-label="开始短剧">
            <div className="creation-composer-stage is-home-mode">
              <div className="creation-empty-composer">{composer}</div>
            </div>
            <LayoutGroup id="drama-empty-suggest">
              <div className="creation-empty-suggest" aria-label="快捷创作入口">
                {starters.map((item) => {
                  const Icon = item.icon;
                  return (
                    <motion.button
                      key={item.title}
                      type="button"
                      className="suggest-card"
                      onClick={() => {
                        setPrompt(item.prompt);
                        window.requestAnimationFrame(() => composerFocusRef.current?.focus());
                      }}
                      onHoverStart={() => setHovered(item.title)}
                      onHoverEnd={() => setHovered(null)}
                      whileHover={reducedMotion ? undefined : { y: -2 }}
                      transition={aceternityMotion.spring.surface}
                    >
                      {hovered === item.title ? (
                        <motion.span layoutId="drama-suggest-hover" className="suggest-card-hover" aria-hidden transition={reducedMotion ? { duration: 0 } : aceternityMotion.spring.surface} />
                      ) : null}
                      <span className="library-icon-tile suggest-icon"><Icon size={18} strokeWidth={2} /></span>
                      <span className="suggest-copy">
                        <strong>{item.title}</strong>
                        <span>{item.hint}</span>
                      </span>
                    </motion.button>
                  );
                })}
              </div>
            </LayoutGroup>
          </section>
        </main>
      ) : (
        <div className="creation-thread-workbench">
          <header className="creation-thread-toolbar">
            <div className="creation-toolbar-shots">
              <span className="creation-rail-trigger"><Film />短剧 Agent</span>
            </div>
            <div className="creation-toolbar-actions">
              <Button size="small" icon={<LayoutGrid className="size-3.5" />} onClick={() => setVideoBoard(true)} data-testid="drama-open-board">打开工坊</Button>
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
