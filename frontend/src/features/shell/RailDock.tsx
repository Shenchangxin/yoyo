/**
 * 拥有：侧栏底部的玻璃图标菜单。底板颜色读主题令牌，字色跟前景。会话入口上的数字只数「需要你」。
 * Buddy 不在这里：它在会话的今天栏里，和今天、任务、记忆并排，要处理的 agent 数挂在那个页签上。
 * 不拥有：切到哪一面。点下去只调用外面传入的回调。
 */
import { useLayoutEffect, useRef, useState } from "react";
import { BookOpen, Clapperboard, FileText, GitBranch, Heart, MessageSquare, Settings } from "lucide-react";
import GlassIcons from "./GlassIcons";
import { useTheme } from "../../lib/theme";
import type { Copy } from "../../lib/copy";
import type { Surface } from "../../lib/protocol";

const DOCK = {
  light: {
    accent: "#3558f5",
    mark: "#c2512a",
    warning: "#a16207",
    success: "#0e7a4b",
    muted: "#5c6472",
    markDeep: "#9a3e20",
    ink: "#1c1f26",
    glass: "#ffffff",
  },
  dark: {
    accent: "#8b9cff",
    mark: "#f2965c",
    warning: "#f5b851",
    success: "#3dd68c",
    muted: "#a3a9b5",
    markDeep: "#d47b44",
    ink: "#f4f5f7",
    glass: "#1a1d24",
  },
};

/** 只接受十六进制。半透明颜色会让玻璃采样跳色。 */
function pickHex(tokens: Record<string, string> | undefined, name: string, fallback: string) {
  const value = tokens?.[name]?.trim() || "";
  return /^#[0-9a-fA-F]{3,8}$/.test(value) ? value : fallback;
}

/** 颜色跟当前明暗一起算出来。不在绘制后再读 CSS，避免切主题时闪一下。 */
function useDockTokens() {
  const { resolved, skin } = useTheme();
  const light = resolved === "light";
  const base = light ? DOCK.light : DOCK.dark;
  const over = skin?.tokens;
  return {
    light,
    tokens: {
      accent: pickHex(over, "--accent", base.accent),
      mark: pickHex(over, "--mark", base.mark),
      warning: pickHex(over, "--warning", base.warning),
      success: pickHex(over, "--success", base.success),
      muted: pickHex(over, "--muted", base.muted),
      markDeep: pickHex(over, "--mark-deep", base.markDeep),
      ink: pickHex(over, "--foreground", base.ink),
      glass: pickHex(over, "--card", base.glass),
    },
  };
}

const DOCK_COUNT = 7;
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

/** RailDock 画出侧栏底部的七个入口。当前面加一圈强调色，Harness 有未提交改动时右上角加一点，会话入口带「需要你」的数字。 */
export function RailDock(props: {
  copy: Copy;
  surface: Surface;
  mark?: boolean;
  needs?: number;
  onChats: () => void;
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
  const items = [
    { id: "sessions", icon: <MessageSquare />, color: tokens.accent, label: label.sessions, onClick: props.onChats, current: props.surface === "agent", badge: needs, badgeLabel: label.needsBadge.replace("{n}", String(needs)) },
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
        frost={0.09}
        shine={light ? 0.58 : 0.78}
        tint={tokens.glass}
        tintOpacity={light ? 0.34 : 0.2}
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
