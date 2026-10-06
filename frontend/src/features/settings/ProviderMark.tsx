import type { ComponentType, SVGProps } from "react";
import AlibabaCloud from "@lobehub/icons/es/AlibabaCloud/components/Mono";
import Aws from "@lobehub/icons/es/Aws/components/Mono";
import AzureColor from "@lobehub/icons/es/Azure/components/Color";
import Claude from "@lobehub/icons/es/Claude/components/Mono";
import ComfyUI from "@lobehub/icons/es/ComfyUI/components/Mono";
import DeepSeek from "@lobehub/icons/es/DeepSeek/components/Mono";
import DoubaoColor from "@lobehub/icons/es/Doubao/components/Color";
import GeminiColor from "@lobehub/icons/es/Gemini/components/Color";
import Grok from "@lobehub/icons/es/Grok/components/Mono";
import Groq from "@lobehub/icons/es/Groq/components/Mono";
import Kimi from "@lobehub/icons/es/Kimi/components/Mono";
import Minimax from "@lobehub/icons/es/Minimax/components/Mono";
import OpenAI from "@lobehub/icons/es/OpenAI/components/Mono";
import OpenRouter from "@lobehub/icons/es/OpenRouter/components/Mono";
import Qwen from "@lobehub/icons/es/Qwen/components/Mono";
import ZAI from "@lobehub/icons/es/ZAI/components/Mono";
import { cn } from "../../lib/utils";

type Glyph = ComponentType<SVGProps<SVGSVGElement> & { size?: number | string; color?: string }>;

type Mark = {
  Icon: Glyph;
  bg: string;
  fg?: string;
  scale?: number;
  painted?: boolean;
};

const ALIAS: Record<string, string> = {
  anthropic: "claude",
  google: "gemini",
  moonshot: "kimi",
  zhipu: "zai",
  glm: "zai",
  chatglm: "zai",
  xai: "grok",
  dashscope: "qwen",
  tongyi: "qwen",
  alibaba: "aliyun",
  alibabacloud: "aliyun",
  wan: "aliyun",
  bytedance: "volcengine",
  doubao: "volcengine",
  seedance: "volcengine",
  seedream: "volcengine",
  ark: "volcengine",
  hailuo: "minimax",
  aws: "s3",
  amazon: "s3",
  "azure-openai": "azure",
  azureai: "azure",
  comfyui: "runninghub",
  openai_compat: "custom",
};

const MARK: Record<string, Mark> = {
  openai: { Icon: OpenAI, bg: "#000", fg: "#fff", scale: 0.74 },
  claude: { Icon: Claude, bg: "#D97757", fg: "#fff", scale: 0.74 },
  gemini: { Icon: GeminiColor, bg: "#fff", painted: true, scale: 0.8 },
  deepseek: { Icon: DeepSeek, bg: "#4D6BFE", fg: "#fff", scale: 0.74 },
  kimi: { Icon: Kimi, bg: "#000", fg: "#fff", scale: 0.6 },
  qwen: { Icon: Qwen, bg: "#615CED", fg: "#fff", scale: 0.74 },
  zai: { Icon: ZAI, bg: "#000", fg: "#fff", scale: 0.62 },
  grok: { Icon: Grok, bg: "#000", fg: "#fff", scale: 0.74 },
  groq: { Icon: Groq, bg: "#F55036", fg: "#fff", scale: 0.74 },
  openrouter: { Icon: OpenRouter, bg: "#111", fg: "#C8FF00", scale: 0.74 },
  azure: { Icon: AzureColor, bg: "#fff", painted: true, scale: 0.7 },
  volcengine: { Icon: DoubaoColor, bg: "#fff", painted: true, scale: 0.78 },
  minimax: { Icon: Minimax, bg: "linear-gradient(90deg, #E2167E, #FE603C)", fg: "#fff", scale: 0.72 },
  aliyun: { Icon: AlibabaCloud, bg: "#FF6A00", fg: "#fff", scale: 0.7 },
  s3: { Icon: Aws, bg: "#222F3E", fg: "#fff", scale: 0.74 },
  runninghub: { Icon: ComfyUI, bg: "#162DD4", fg: "#F0FF41", scale: 0.6 },
  cas: { Icon: CasGlyph, bg: "#3F3F46", fg: "#fff", scale: 0.7 },
  search: { Icon: SearchGlyph, bg: "#2563EB", fg: "#fff", scale: 0.68 },
  otel: { Icon: OtelGlyph, bg: "#425CC7", fg: "#fff", scale: 0.7 },
  custom: { Icon: CustomGlyph, bg: "#52525B", fg: "#fff", scale: 0.7 },
};

export function ProviderMark({ id, className }: { id: string; className?: string }) {
  const key = resolveProviderId(id);
  const mark = MARK[key] || MARK.custom;
  const scale = mark.scale ?? 0.74;
  const light = isLightFill(mark.bg);
  const Icon = mark.Icon;
  return (
    <span
      className={cn(
        "relative grid size-9 shrink-0 place-items-center overflow-hidden rounded-[22%] text-white",
        className,
      )}
      style={{
        background: mark.bg,
        color: mark.fg,
        boxShadow: light
          ? "inset 0 0 0 1px rgba(15,15,15,0.08), 0 0.5px 0.5px rgba(15,15,15,0.04)"
          : "inset 0 0 0 1px rgba(255,255,255,0.12)",
      }}
      aria-hidden
    >
      <Icon
        size={`${Math.round(scale * 100)}%`}
        color={mark.painted ? undefined : mark.fg}
        className="block"
      />
    </span>
  );
}

export function resolveProviderId(id: string): string {
  const raw = (id || "").trim().toLowerCase().replace(/[\s_]+/g, "-");
  if (!raw) return "custom";
  return ALIAS[raw] || (MARK[raw] ? raw : "custom");
}

function isLightFill(bg: string): boolean {
  const s = bg.trim().toLowerCase();
  return s === "#fff" || s === "#ffffff" || s === "white";
}

function CasGlyph({ size = "1em", color = "currentColor", ...rest }: SVGProps<SVGSVGElement> & { size?: number | string }) {
  return (
    <svg viewBox="0 0 24 24" width={size} height={size} fill="none" {...rest}>
      <path d="M4.8 7.2 12 4.4l7.2 2.8v9.6L12 19.6 4.8 16.8V7.2Z" stroke={color} strokeWidth="1.7" strokeLinejoin="round" />
      <path d="M12 4.4v15.2M4.8 7.2 12 10l7.2-2.8" stroke={color} strokeWidth="1.7" strokeLinejoin="round" />
    </svg>
  );
}

function SearchGlyph({ size = "1em", color = "currentColor", ...rest }: SVGProps<SVGSVGElement> & { size?: number | string }) {
  return (
    <svg viewBox="0 0 24 24" width={size} height={size} fill="none" {...rest}>
      <circle cx="10.5" cy="10.5" r="5.4" stroke={color} strokeWidth="1.8" />
      <path d="M14.8 14.8 19 19" stroke={color} strokeWidth="1.8" strokeLinecap="round" />
    </svg>
  );
}

function OtelGlyph({ size = "1em", color = "currentColor", ...rest }: SVGProps<SVGSVGElement> & { size?: number | string }) {
  return (
    <svg viewBox="0 0 24 24" width={size} height={size} fill={color} {...rest}>
      <circle cx="12" cy="12" r="2.1" />
      <path d="M12 4.2v3.2M12 16.6v3.2M4.2 12h3.2M16.6 12h3.2M6.5 6.5l2.3 2.3M15.2 15.2l2.3 2.3M17.5 6.5l-2.3 2.3M8.8 15.2l-2.3 2.3" stroke={color} strokeWidth="1.7" strokeLinecap="round" fill="none" />
    </svg>
  );
}

function CustomGlyph({ size = "1em", color = "currentColor", ...rest }: SVGProps<SVGSVGElement> & { size?: number | string }) {
  return (
    <svg viewBox="0 0 24 24" width={size} height={size} fill="none" {...rest}>
      <path d="M12 3.4 20.4 8.2v7.6L12 20.6 3.6 15.8V8.2L12 3.4Z" stroke={color} strokeWidth="1.7" strokeLinejoin="round" />
      <circle cx="12" cy="12" r="2.15" fill={color} />
    </svg>
  );
}
