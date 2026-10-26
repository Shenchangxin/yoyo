/**
 * 拥有：服务商图标和别名到图标的解析。
 * 不拥有：服务商表单。只绘制，不调用 api。
 */
import type { ComponentType, SVGProps } from "react";
import AlibabaCloud from "@lobehub/icons/es/AlibabaCloud/components/Mono";
import AssemblyAIColor from "@lobehub/icons/es/AssemblyAI/components/Color";
import Aws from "@lobehub/icons/es/Aws/components/Mono";
import AzureColor from "@lobehub/icons/es/Azure/components/Color";
import Claude from "@lobehub/icons/es/Claude/components/Mono";
import ComfyUI from "@lobehub/icons/es/ComfyUI/components/Mono";
import DeepSeek from "@lobehub/icons/es/DeepSeek/components/Mono";
import DoubaoColor from "@lobehub/icons/es/Doubao/components/Color";
import ElevenLabs from "@lobehub/icons/es/ElevenLabs/components/Mono";
import GeminiColor from "@lobehub/icons/es/Gemini/components/Color";
import Grok from "@lobehub/icons/es/Grok/components/Mono";
import Groq from "@lobehub/icons/es/Groq/components/Mono";
import HunyuanColor from "@lobehub/icons/es/Hunyuan/components/Color";
import JimengColor from "@lobehub/icons/es/Jimeng/components/Color";
import Kimi from "@lobehub/icons/es/Kimi/components/Mono";
import KlingColor from "@lobehub/icons/es/Kling/components/Color";
import KolorsColor from "@lobehub/icons/es/Kolors/components/Color";
import Minimax from "@lobehub/icons/es/Minimax/components/Mono";
import OpenAI from "@lobehub/icons/es/OpenAI/components/Mono";
import OpenRouter from "@lobehub/icons/es/OpenRouter/components/Mono";
import Qwen from "@lobehub/icons/es/Qwen/components/Mono";
import Runway from "@lobehub/icons/es/Runway/components/Mono";
import SkyworkColor from "@lobehub/icons/es/Skywork/components/Color";
import SparkColor from "@lobehub/icons/es/Spark/components/Color";
import StabilityColor from "@lobehub/icons/es/Stability/components/Color";
import Suno from "@lobehub/icons/es/Suno/components/Mono";
import TencentCloudColor from "@lobehub/icons/es/TencentCloud/components/Color";
import ViduColor from "@lobehub/icons/es/Vidu/components/Color";
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
  tavily: "search",
  brave: "search",
  serper: "search",
  bocha: "search",
  microsoft: "azure",
  kuaishou: "kling",
  lyria: "gemini",
  iflytek: "spark",
  xfyun: "spark",
  xunfei: "spark",
  mureka: "skywork",
  skywork: "skywork",
  hunyuan: "hunyuan",
  metaso: "search",
  cos: "tencent",
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
  elevenlabs: { Icon: ElevenLabs, bg: "#000", fg: "#fff", scale: 0.7 },
  suno: { Icon: Suno, bg: "#000", fg: "#fff", scale: 0.7 },
  stability: { Icon: StabilityColor, bg: "#fff", painted: true, scale: 0.74 },
  runway: { Icon: Runway, bg: "#111", fg: "#fff", scale: 0.7 },
  kling: { Icon: KlingColor, bg: "#fff", painted: true, scale: 0.74 },
  assemblyai: { Icon: AssemblyAIColor, bg: "#fff", painted: true, scale: 0.74 },
  pexels: { Icon: PexelsGlyph, bg: "#05A081", fg: "#fff", scale: 0.7 },
  pixabay: { Icon: PixabayGlyph, bg: "#2EC66D", fg: "#fff", scale: 0.7 },
  unsplash: { Icon: UnsplashGlyph, bg: "#111", fg: "#fff", scale: 0.68 },
  giphy: { Icon: GiphyGlyph, bg: "#111", fg: "#fff", scale: 0.7 },
  deepgram: { Icon: DeepgramGlyph, bg: "#0B0B0F", fg: "#13EF93", scale: 0.7 },
  jimeng: { Icon: JimengColor, bg: "#fff", painted: true, scale: 0.74 },
  vidu: { Icon: ViduColor, bg: "#fff", painted: true, scale: 0.74 },
  kolors: { Icon: KolorsColor, bg: "#fff", painted: true, scale: 0.74 },
  hunyuan: { Icon: HunyuanColor, bg: "#fff", painted: true, scale: 0.74 },
  tencent: { Icon: TencentCloudColor, bg: "#fff", painted: true, scale: 0.7 },
  spark: { Icon: SparkColor, bg: "#fff", painted: true, scale: 0.74 },
  skywork: { Icon: SkyworkColor, bg: "#fff", painted: true, scale: 0.74 },
  custom: { Icon: CustomGlyph, bg: "#52525B", fg: "#fff", scale: 0.7 },
};

/** ProviderMark 按服务商编号画出图标。id 传给 resolveProviderId。className 传给 cn。 */
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

/** resolveProviderId 把别名收成图标表里的编号，没有就用 custom。id 为假时用空字符串，去掉首尾空白，转成小写，做替换。 */
export function resolveProviderId(id: string): string {
  const raw = (id || "").trim().toLowerCase().replace(/[\s_]+/g, "-");
  if (!raw) return "custom";
  return ALIAS[raw] || (MARK[raw] ? raw : "custom");
}

/** 背景是不是浅色。bg 去掉首尾空白，转成小写后，与 #fff、#ffffff、white 比较是否相等。 */
function isLightFill(bg: string): boolean {
  const s = bg.trim().toLowerCase();
  return s === "#fff" || s === "#ffffff" || s === "white";
}

/** 画 CAS 的图标。size 写到 svg 的 width。color 写到 path 的 stroke。rest 展开到 svg。 */
function CasGlyph({ size = "1em", color = "currentColor", ...rest }: SVGProps<SVGSVGElement> & { size?: number | string }) {
  return (
    <svg viewBox="0 0 24 24" width={size} height={size} fill="none" {...rest}>
      <path d="M4.8 7.2 12 4.4l7.2 2.8v9.6L12 19.6 4.8 16.8V7.2Z" stroke={color} strokeWidth="1.7" strokeLinejoin="round" />
      <path d="M12 4.4v15.2M4.8 7.2 12 10l7.2-2.8" stroke={color} strokeWidth="1.7" strokeLinejoin="round" />
    </svg>
  );
}

/** 画搜索的图标。size 写到 svg 的 width。color 写到 circle 的 stroke。rest 展开到 svg。 */
function SearchGlyph({ size = "1em", color = "currentColor", ...rest }: SVGProps<SVGSVGElement> & { size?: number | string }) {
  return (
    <svg viewBox="0 0 24 24" width={size} height={size} fill="none" {...rest}>
      <circle cx="10.5" cy="10.5" r="5.4" stroke={color} strokeWidth="1.8" />
      <path d="M14.8 14.8 19 19" stroke={color} strokeWidth="1.8" strokeLinecap="round" />
    </svg>
  );
}

/** 画 OpenTelemetry 的图标。size 写到 svg 的 width。color 写到 svg 的 fill。rest 展开到 svg。 */
function OtelGlyph({ size = "1em", color = "currentColor", ...rest }: SVGProps<SVGSVGElement> & { size?: number | string }) {
  return (
    <svg viewBox="0 0 24 24" width={size} height={size} fill={color} {...rest}>
      <circle cx="12" cy="12" r="2.1" />
      <path d="M12 4.2v3.2M12 16.6v3.2M4.2 12h3.2M16.6 12h3.2M6.5 6.5l2.3 2.3M15.2 15.2l2.3 2.3M17.5 6.5l-2.3 2.3M8.8 15.2l-2.3 2.3" stroke={color} strokeWidth="1.7" strokeLinecap="round" fill="none" />
    </svg>
  );
}

/** 画自定义服务的图标。size 写到 svg 的 width。color 写到 path 的 stroke。rest 展开到 svg。 */
function CustomGlyph({ size = "1em", color = "currentColor", ...rest }: SVGProps<SVGSVGElement> & { size?: number | string }) {
  return (
    <svg viewBox="0 0 24 24" width={size} height={size} fill="none" {...rest}>
      <path d="M12 3.4 20.4 8.2v7.6L12 20.6 3.6 15.8V8.2L12 3.4Z" stroke={color} strokeWidth="1.7" strokeLinejoin="round" />
      <circle cx="12" cy="12" r="2.15" fill={color} />
    </svg>
  );
}

/** 画 Pexels 的图标。size 写到 svg 的 width。color 写到 rect 的 stroke。rest 展开到 svg。 */
function PexelsGlyph({ size = "1em", color = "currentColor", ...rest }: SVGProps<SVGSVGElement> & { size?: number | string }) {
  return (
    <svg viewBox="0 0 24 24" width={size} height={size} fill="none" {...rest}>
      <rect x="4.2" y="6.2" width="15.6" height="11.6" rx="2.2" stroke={color} strokeWidth="1.7" />
      <circle cx="12" cy="12" r="2.6" stroke={color} strokeWidth="1.7" />
    </svg>
  );
}

/** 画 Pixabay 的图标。size 写到 svg 的 width。color 写到 path 的 stroke。rest 展开到 svg。 */
function PixabayGlyph({ size = "1em", color = "currentColor", ...rest }: SVGProps<SVGSVGElement> & { size?: number | string }) {
  return (
    <svg viewBox="0 0 24 24" width={size} height={size} fill="none" {...rest}>
      <path d="M5 16.4 9.4 8.6 12 13.2 14.6 8.6 19 16.4" stroke={color} strokeWidth="1.7" strokeLinejoin="round" />
    </svg>
  );
}

/** 画 Unsplash 的图标。size 写到 svg 的 width。color 写到 svg 的 fill。rest 展开到 svg。 */
function UnsplashGlyph({ size = "1em", color = "currentColor", ...rest }: SVGProps<SVGSVGElement> & { size?: number | string }) {
  return (
    <svg viewBox="0 0 24 24" width={size} height={size} fill={color} {...rest}>
      <path d="M8.4 7.2h7.2v3.2H14V9.2h-4v5.6h4v-1.2h1.6v3.2H8.4V7.2Z" />
    </svg>
  );
}

/** 画 Giphy 的图标。size 写到 svg 的 width。color 写到 path 的 fill。rest 展开到 svg。 */
function GiphyGlyph({ size = "1em", color = "currentColor", ...rest }: SVGProps<SVGSVGElement> & { size?: number | string }) {
  return (
    <svg viewBox="0 0 24 24" width={size} height={size} fill="none" {...rest}>
      <path d="M7 6.4h6.2v2.2H9.2v6.8H7V6.4Zm6.2 2.2H17V17.6h-3.8v-2.2H15v-4.6h-1.8V8.6Z" fill={color} />
    </svg>
  );
}

/** 画 Deepgram 的图标。size 写到 svg 的 width。color 写到 path 的 stroke。rest 展开到 svg。 */
function DeepgramGlyph({ size = "1em", color = "currentColor", ...rest }: SVGProps<SVGSVGElement> & { size?: number | string }) {
  return (
    <svg viewBox="0 0 24 24" width={size} height={size} fill="none" {...rest}>
      <path d="M5 12h2.2M9.2 7.4v9.2M12.6 5.6v12.8M16 7.4v9.2M19 12h-2.2" stroke={color} strokeWidth="1.7" strokeLinecap="round" />
    </svg>
  );
}
