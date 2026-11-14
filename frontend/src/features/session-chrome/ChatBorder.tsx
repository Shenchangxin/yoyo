/**
 * 拥有：对话输入框外圈的电流。颜色读文档根上的 --accent，明暗跟当前主题。
 * 电流只表示"正在干活"：调用方在运行时传 active。窗口失焦 30 秒后停下，回到窗口再接上。
 * 不拥有：框里的草稿和发送。WebGL 起不来时只留下原来的输入框。
 */
import { useEffect, useRef, useState, type ReactNode } from "react";
import ElectricBorder from "./ElectricBorder";
import { useTheme } from "../../lib/theme";

const PAD = 3;
const FALLBACK = { dark: "#6e85ff", light: "#3b5bff" };
const IDLE_AFTER_BLUR_MS = 30_000;

/** 窗口在前台，或者离开还不到 30 秒。 */
function useWindowAttention(): boolean {
  const [attentive, setAttentive] = useState(true);
  useEffect(() => {
    let timer = 0;
    const onFocus = () => {
      window.clearTimeout(timer);
      setAttentive(true);
    };
    const onBlur = () => {
      window.clearTimeout(timer);
      timer = window.setTimeout(() => setAttentive(false), IDLE_AFTER_BLUR_MS);
    };
    window.addEventListener("focus", onFocus);
    window.addEventListener("blur", onBlur);
    return () => {
      window.clearTimeout(timer);
      window.removeEventListener("focus", onFocus);
      window.removeEventListener("blur", onBlur);
    };
  }, []);
  return attentive;
}

/** 读文档根上的强调色。皮肤和调色板改到根上，这里跟着变。 */
function useAccent(fallback: string) {
  const { resolved, palette, skinId } = useTheme();
  const [color, setColor] = useState(fallback);
  useEffect(() => {
    const read = () => {
      const next = getComputedStyle(document.documentElement).getPropertyValue("--accent").trim();
      setColor(next || fallback);
    };
    read();
    const observer = new MutationObserver(read);
    observer.observe(document.documentElement, {
      attributes: true,
      attributeFilter: ["class", "style", "data-palette", "data-skin"],
    });
    return () => observer.disconnect();
  }, [fallback, resolved, palette, skinId]);
  return color;
}

/** 量内容自己的圆角。胶囊会报一个很大的半径，描边时再夹回半高。 */
function useCorner(explicit: number | undefined) {
  const fitRef = useRef<HTMLDivElement>(null);
  const { resolved, palette, skinId } = useTheme();
  const [radius, setRadius] = useState(explicit ?? 20);
  useEffect(() => {
    if (explicit != null) {
      setRadius(explicit);
      return;
    }
    const node = fitRef.current;
    if (!node) return;
    const read = () => {
      const targets = [node, ...Array.from(node.querySelectorAll<HTMLElement>("*"))].slice(0, 8);
      for (const target of targets) {
        const value = parseFloat(getComputedStyle(target).borderTopLeftRadius);
        if (value > 0) {
          setRadius(value);
          return;
        }
      }
    };
    read();
    const observer = new ResizeObserver(read);
    observer.observe(node);
    return () => observer.disconnect();
  }, [explicit, resolved, palette, skinId]);
  return { fitRef, radius };
}

/** ChatBorder 包住一个对话输入框。active 为假时只剩输入框自己的细线。tone 只在画面本身是暗的（通话层）时盖过页面主题。 */
export function ChatBorder(props: {
  children: ReactNode;
  active?: boolean;
  tone?: "light" | "dark";
  radius?: number;
  className?: string;
}) {
  const { resolved } = useTheme();
  const light = (props.tone ?? resolved) === "light";
  const color = useAccent(light ? FALLBACK.light : FALLBACK.dark);
  const { fitRef, radius } = useCorner(props.radius);
  const attentive = useWindowAttention();
  return (
    <ElectricBorder
      className={props.className || ""}
      color={color}
      theme={light ? "light" : "dark"}
      borderRadius={radius + PAD}
      speed={0.62}
      chaos={0.16}
      spread={0.22}
      strands={2}
      thickness={1.08}
      glow={light ? 0.18 : 0.34}
      intensity={light ? 0.62 : 0.78}
      arcs={0.22}
      flicker={0.14}
      offset={1.5}
      active={!!props.active && attentive}
      intro
      interactive
      cursorRadius={72}
      style={{ padding: PAD }}
    >
      <div ref={fitRef} className="chat-border-fit">
        {props.children}
      </div>
    </ElectricBorder>
  );
}
