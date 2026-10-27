/**
 * 拥有：对话输入框外圈的电流。颜色读文档根上的 --accent，明暗跟当前主题。
 * 不拥有：框里的草稿和发送。WebGL 起不来时只留下原来的输入框。
 */
import { useEffect, useRef, useState, type ReactNode } from "react";
import ElectricBorder from "./ElectricBorder";
import { useTheme } from "../../lib/theme";

const PAD = 3;
const FALLBACK = { dark: "#5c99d6", light: "#0169cc" };

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

/** ChatBorder 包住一个对话输入框。tone 只在画面本身是暗的（通话层）时盖过页面主题。 */
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
  return (
    <ElectricBorder
      className={props.className || ""}
      color={color}
      theme={light ? "light" : "dark"}
      borderRadius={radius + PAD}
      speed={0.75}
      chaos={0.28}
      spread={0.26}
      strands={2}
      thickness={1.35}
      glow={light ? 0.28 : 0.5}
      intensity={light ? 0.7 : 0.85}
      arcs={0.35}
      flicker={0.25}
      offset={2}
      active={props.active ?? true}
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
