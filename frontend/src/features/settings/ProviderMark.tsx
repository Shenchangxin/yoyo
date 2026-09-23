import type { JSX } from "react";
import { cn } from "../../lib/utils";

export function ProviderMark({ id, className }: { id: string; className?: string }) {
  const glyph = GLYPH[id] || GLYPH.custom;
  return (
    <span
      className={cn("grid size-9 shrink-0 place-items-center rounded-[10px] text-white shadow-[inset_0_0_0_1px_rgba(255,255,255,0.12)]", className)}
      style={{ background: glyph.bg }}
      aria-hidden
    >
      {glyph.svg}
    </span>
  );
}

const GLYPH: Record<string, { bg: string; svg: JSX.Element }> = {
  openai: { bg: "#10A37F", svg: (
    <svg viewBox="0 0 24 24" className="size-5" fill="currentColor">
      <path d="M12.8 3.1c.9-1.5 3.1-1.5 4 0l1.6 2.8c.2.3.5.6.9.7l3.1.4c1.7.3 2.4 2.3 1.1 3.5l-2.3 2.2c-.3.3-.4.7-.3 1.1l.6 3.1c.3 1.7-1.4 3-2.9 2.2l-2.8-1.5c-.4-.2-.8-.2-1.2 0l-2.8 1.5c-1.5.8-3.2-.5-2.9-2.2l.6-3.1c.1-.4 0-.8-.3-1.1L6.9 10.5C5.6 9.3 6.3 7.3 8 7l3.1-.4c.4-.1.7-.4.9-.7l.8-1.8Z" />
    </svg>
  ) },
  claude: { bg: "#D97757", svg: (
    <svg viewBox="0 0 24 24" className="size-5" fill="currentColor">
      <path d="M13.2 3.2 15 9.4l5.8.6-4.4 4 1.3 5.7L13 17.2 8.3 19.7l1.3-5.7-4.4-4 5.8-.6 1.8-6.2 0.4-1Z" />
    </svg>
  ) },
  gemini: { bg: "#4285F4", svg: (
    <svg viewBox="0 0 24 24" className="size-5" fill="currentColor">
      <path d="M12 2.5 14.2 9.8 21.5 12 14.2 14.2 12 21.5 9.8 14.2 2.5 12 9.8 9.8Z" />
    </svg>
  ) },
  deepseek: { bg: "#4D6BFE", svg: (
    <svg viewBox="0 0 24 24" className="size-5" fill="none" stroke="currentColor" strokeWidth="1.8">
      <path d="M5 15c2.2 3.2 6 5 10.2 4.2 3.4-.6 6-3.4 6.6-6.8.4-2.6-.4-5.2-2.2-7.1" />
      <circle cx="9.2" cy="9.4" r="1.2" fill="currentColor" stroke="none" />
      <path d="M4.8 10.5c2.8-3 7.4-4 11.2-2.2" />
    </svg>
  ) },
  kimi: { bg: "#1C1C1C", svg: (
    <svg viewBox="0 0 24 24" className="size-5" fill="currentColor">
      <path d="M17.2 5.2a7.4 7.4 0 1 0 1.6 11.4 8.2 8.2 0 0 1-8.6-8.8 8.2 8.2 0 0 1 7-2.6Z" />
    </svg>
  ) },
  qwen: { bg: "#615CED", svg: (
    <svg viewBox="0 0 24 24" className="size-5" fill="none" stroke="currentColor" strokeWidth="2">
      <circle cx="11" cy="11" r="6.2" />
      <path d="M15.4 15.4 20 20" strokeLinecap="round" />
    </svg>
  ) },
  zai: { bg: "#102A43", svg: (
    <svg viewBox="0 0 24 24" className="size-5" fill="currentColor">
      <path d="M6 6h12l-9.4 8.4H18V18H6l9.4-8.4H6V6Z" />
    </svg>
  ) },
  grok: { bg: "#111111", svg: (
    <svg viewBox="0 0 24 24" className="size-5" fill="currentColor">
      <path d="M5 5.5h4.2l3 4.2 3-4.2H19L13.6 12 19 18.5h-4.2l-3-4.3-3 4.3H5L10.4 12Z" />
    </svg>
  ) },
  groq: { bg: "#F55040", svg: (
    <svg viewBox="0 0 24 24" className="size-5" fill="currentColor">
      <path d="M13.2 3 6 13.2h5.1L10 21l8.4-11.6h-5.4L13.2 3Z" />
    </svg>
  ) },
  openrouter: { bg: "#6566F1", svg: (
    <svg viewBox="0 0 24 24" className="size-5" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round">
      <path d="M7 12h10M14 8l4 4-4 4M10 16 6 12l4-4" />
    </svg>
  ) },
  azure: { bg: "#0078D4", svg: (
    <svg viewBox="0 0 24 24" className="size-5" fill="currentColor">
      <path d="M10.4 4 4 20h5.2l1.8-4.4h6.4L10.4 4Zm3.2 3.6 5.4 12.4H22L16.4 7.6h-2.8Z" />
    </svg>
  ) },
  volcengine: { bg: "#1664FF", svg: (
    <svg viewBox="0 0 24 24" className="size-5" fill="currentColor">
      <path d="M12 3.2 4.8 16.8h4.1L12 10.6l3.1 6.2h4.1L12 3.2Zm-5.4 15.1L12 20.8l5.4-2.5H6.6Z" />
    </svg>
  ) },
  minimax: { bg: "#FF5A36", svg: (
    <svg viewBox="0 0 24 24" className="size-5" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round">
      <path d="M4.5 14.5c2-4 4.2-6.5 7.5-6.5s5.5 2.5 7.5 6.5" />
      <path d="M7 17c1.4-2.4 2.8-3.6 5-3.6s3.6 1.2 5 3.6" />
      <circle cx="12" cy="7.2" r="1.2" fill="currentColor" stroke="none" />
    </svg>
  ) },
  aliyun: { bg: "#FF6A00", svg: (
    <svg viewBox="0 0 24 24" className="size-5" fill="currentColor">
      <path d="M7.2 16.2h9.6c2.2 0 4-1.8 4-4 0-1.9-1.3-3.5-3.1-3.9A5.1 5.1 0 0 0 7.4 9.2 3.7 3.7 0 0 0 4 12.8c0 1.9 1.5 3.4 3.2 3.4Z" />
    </svg>
  ) },
  custom: { bg: "#52525B", svg: (
    <svg viewBox="0 0 24 24" className="size-5" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinejoin="round">
      <path d="M8 4h4l2 3h6v5l-3 2 3 2v4H14l-2 3H8l-2-3H4v-4l3-2-3-2V7h2L8 4Z" />
    </svg>
  ) },
};
