/**
 * 拥有：侧栏底部的玻璃图标菜单。底板颜色读主题令牌，字色跟前景。会话入口上的数字只数「需要你」；
 * Buddy 入口上的数字数要处理的 agent，有 agent 在干活时右下角亮一个点（读 Buddy 仓库）。
 * 不拥有：切到哪一面。点下去只调用外面传入的回调。
 */
import { useEffect, useLayoutEffect, useRef, useState } from "react";
import { BookOpen, Bot, Clapperboard, FileText, GitBranch, Heart, MessageSquare, Settings } from "lucide-react";
import GlassIcons from "./GlassIcons";
import { useTheme } from "../../lib/theme";
import type { Copy } from "../../lib/copy";
import type { Surface } from "../../lib/protocol";
import { anyWorking, attentionCount, fill } from "../buddy/buddy-model";
import { useBuddy } from "../buddy/buddy-store";

const FALLBACK = {
  light: {
    accent: "#0169cc",
    mark: "#c2663d",
    warning: "#a07a2a",
    success: "#1f7a45",
    muted: "#6f6e68",
    markDeep: "#9a4e2e",
    ink: "#1c1c1a",
    glass: "#f1f1ee",
  },
  dark: {
    accent: "#5c99d6",
    mark: "#d4896a",
    warning: "#d4b06a",
    success: "#74b48a",
    muted: "#8c8b85",
    markDeep: "#b56545",
    ink: "#f4f4f0",
    glass: "#222220",
  },
};

/** 读文档根上的颜色。皮肤和调色板写在根上，这里跟着变。 */
function useDockTokens() {
  const { resolved, palette, skinId } = useTheme();
  const light = resolved === "light";
  const [tokens, setTokens] = useState(light ? FALLBACK.light : FALLBACK.dark);
  useEffect(() => {
    const style = getComputedStyle(document.documentElement);
    const read = (name: string, fallback: string) => style.getPropertyValue(name).trim() || fallback;
    const base = light ? FALLBACK.light : FALLBACK.dark;
    setTokens({
      accent: read("--accent", base.accent),
      mark: read("--mark", base.mark),
      warning: read("--warning", base.warning),
      success: read("--success", base.success),
      muted: read("--muted", base.muted),
      markDeep: read("--mark-deep", base.markDeep),
      ink: read("--foreground", base.ink),
      glass: read("--sidebar", base.glass),
    });
  }, [light, palette, skinId]);
  return { light, tokens };
}

const DOCK_COUNT = 8;
const DOCK_MAX = 44;
const DOCK_MIN = 18;
/** 间距大约是图标边长的这个比例，宽栏时图标到顶后间距继续拉开。 */
const DOCK_GAP_RATIO = 0.28;
/** 按侧栏内容宽度算出同一行里的图标边长。间距交给两端对齐去分。 */
function dockSize(width: number) {
  if (width <= 0) return 28;
  const ideal = width / (DOCK_COUNT + (DOCK_COUNT - 1) * DOCK_GAP_RATIO);
  let size = Math.round(Math.min(DOCK_MAX, Math.max(DOCK_MIN, ideal)));
  if (size * DOCK_COUNT > width) size = Math.max(16, Math.floor(width / DOCK_COUNT));
  return size;
}

/** 进场只在这次打开后的第一次挂载上播。侧栏之后再挂，不再从透明长出来。 */
let dockIntroPlayed = false;

/** 按这一行的实际宽度算边长。会话页和模块栏都铺满自己的侧栏，边长跟整栏走。 */
function useDockSize(rowRef: { current: HTMLDivElement | null }) {
  const [size, setSize] = useState(28);
  useLayoutEffect(() => {
    const row = rowRef.current;
    if (!row) return;
    const fit = () => {
      const width = row.clientWidth;
      if (width <= 0) return;
      const next = dockSize(width);
      setSize((prev) => (prev === next ? prev : next));
    };
    fit();
    const observer = new ResizeObserver(fit);
    observer.observe(row);
    window.addEventListener("resize", fit);
    return () => {
      observer.disconnect();
      window.removeEventListener("resize", fit);
    };
  }, [rowRef]);
  return size;
}

/** RailDock 画出侧栏底部的八个入口。当前面加一圈强调色，Harness 有未提交改动时右上角加一点，会话入口带「需要你」的数字，
 *  Buddy 入口带要处理的 agent 数。 */
export function RailDock(props: {
  copy: Copy;
  surface: Surface;
  mark?: boolean;
  needs?: number;
  onChats: () => void;
  onBuddy: () => void;
  onSkills: () => void;
  onVideo: () => void;
  onPages: () => void;
  onIdol: () => void;
  onHarness: () => void;
  onSettings: () => void;
}) {
  const { light, tokens } = useDockTokens();
  const rowRef = useRef<HTMLDivElement>(null);
  const size = useDockSize(rowRef);
  const [intro] = useState(() => {
    if (dockIntroPlayed) return false;
    dockIntroPlayed = true;
    return true;
  });
  const label = props.copy.rail;
  const needs = props.needs || 0;
  const buddyNeeds = useBuddy((s) => attentionCount(s.list));
  const buddyLive = useBuddy((s) => anyWorking(s.list));
  const items = [
    { id: "sessions", icon: <MessageSquare />, color: tokens.accent, label: label.sessions, onClick: props.onChats, current: props.surface === "agent", badge: needs, badgeLabel: label.needsBadge.replace("{n}", String(needs)) },
    { id: "buddy", icon: <Bot />, color: tokens.success, label: label.buddy, onClick: props.onBuddy, current: props.surface === "buddy", badge: buddyNeeds, badgeLabel: fill(label.buddyBadge, { n: buddyNeeds }), className: buddyLive ? "has-live" : "" },
    { id: "skills", icon: <BookOpen />, color: tokens.mark, label: label.skills, onClick: props.onSkills, current: props.surface === "skills" },
    { id: "video", icon: <Clapperboard />, color: tokens.warning, label: label.video, onClick: props.onVideo, current: props.surface === "video" },
    { id: "pages", icon: <FileText />, color: tokens.muted, label: label.pages, onClick: props.onPages, current: props.surface === "pages" },
    { id: "idol", icon: <Heart />, color: tokens.markDeep, label: label.idol, onClick: props.onIdol, current: props.surface === "idol" },
    { id: "harness", icon: <GitBranch />, color: tokens.success, label: label.harness, onClick: props.onHarness, current: props.surface === "harness", className: props.mark ? "has-mark" : "" },
    { id: "settings", icon: <Settings />, color: tokens.muted, label: label.settings, onClick: props.onSettings, current: props.surface === "settings" },
  ];
  return (
    <div className="rail-glass-host" ref={rowRef}>
      <GlassIcons
        className="rail-glass"
        items={items.map((item) => ({
          ...item,
          className: [item.current ? "is-current" : "", "className" in item ? item.className : ""].filter(Boolean).join(" "),
        }))}
        size={size}
        gap={0}
        columns={DOCK_COUNT}
        roundness={0.62}
        refraction={0.45}
        bevel={0.45}
        dispersion={0}
        frost={0.08}
        shine={light ? 0.5 : 0.72}
        tint={tokens.glass}
        tintOpacity={light ? 0.55 : 0.28}
        iconColor={tokens.ink}
        plate="tilt"
        tilt={8}
        spread={0.18}
        hover="press"
        parallax={0.35}
        labels="hover"
        intro={intro}
        theme={light ? "light" : "dark"}
      />
    </div>
  );
}
