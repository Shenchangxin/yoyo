/**
 * 拥有：生成服务商、画布通道和本机工具的设置。
 * 不拥有：偶像服务。保存、安装和探测会调用 api 改状态。
 */
import { useEffect, useMemo, useState, type ReactNode } from "react";
import { Download, Plus } from "lucide-react";
import { toast } from "sonner";
import { useCopy } from "../../lib/i18n";
import * as api from "../../lib/client";
import { Button } from "../../components/ui/button";
import { Input } from "../../components/ui/input";
import { Switch } from "../../components/ui/switch";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "../../components/ui/select";
import { ConfirmDialog } from "../ConfirmDialog";
import { ProgressHairline } from "../../components/ui/progress-hairline";
import { cn } from "../../lib/utils";
import {
  CONTROL_LG,
  SettingEmpty,
  SettingRow,
  SettingSection,
  SettingSegmented,
} from "./SettingChrome";
import { ProviderMark } from "./ProviderMark";

type Kind = "image" | "video" | "tts" | "music" | "matting" | "stock" | "asr" | "storage" | "search" | "workflow";

type MediaProvider = {
  id: string;
  service_type: Kind | string;
  provider: string;
  name: string;
  base_url: string;
  model: string;
  models: string;
  settings?: string;
  is_default?: boolean;
  is_active?: boolean;
  has_key?: boolean;
  priority?: number;
};

type Draft = {
  id?: string;
  service_type: Kind;
  provider: string;
  name: string;
  base_url: string;
  model: string;
  models: string;
  api_key: string;
  is_default: boolean;
  is_active: boolean;
  has_key?: boolean;
  bucket?: string;
  region?: string;
  path_style?: boolean;
  cdn_base_url?: string;
};

const VENDOR_LABEL: Record<string, string> = {
  volcengine: "Volcengine",
  openai: "OpenAI",
  gemini: "Gemini",
  minimax: "MiniMax",
  aliyun: "Alibaba Cloud",
  s3: "S3",
  cas: "CAS",
  runninghub: "RunningHub",
  search: "Search",
  jina: "Jina",
  exa: "Exa",
  duckduckgo: "DuckDuckGo",
  tavily: "Tavily",
  brave: "Brave",
  serper: "Serper",
  bocha: "Bocha",
  otel: "OpenTelemetry",
  custom: "Custom",
  elevenlabs: "ElevenLabs",
  azure: "Azure",
  pexels: "Pexels",
  pixabay: "Pixabay",
  unsplash: "Unsplash",
  giphy: "Giphy",
  groq: "Groq",
  deepgram: "Deepgram",
  assemblyai: "AssemblyAI",
  stability: "Stability",
  suno: "Suno",
  kling: "Kling",
  runway: "Runway",
  jimeng: "Jimeng",
  vidu: "Vidu",
  tencent: "Tencent",
  hunyuan: "Hunyuan",
  zhipu: "Zhipu",
  kolors: "Kolors",
  spark: "Spark",
  mureka: "Mureka",
  removebg: "remove.bg",
  photoroom: "Photoroom",
  metaso: "Metaso",
};

/** 这种服务要不要填模型列表。kind 是 image、video、tts、music、matting 或 asr 时返回真。 */
function usesModels(kind: Kind): boolean {
  return kind === "image" || kind === "video" || kind === "tts" || kind === "music" || kind === "matting" || kind === "asr";
}

/** 把服务种类收成面板用的名字。p 读取 service_type、serviceType，用 typeof 看类型，作为分支结果。 */
function serviceKind(p: { service_type?: string; serviceType?: string } | string): string {
  const raw = typeof p === "string" ? p : String(p.service_type || p.serviceType || "");
  const s = raw.toLowerCase().trim();
  if (s === "speech" || s === "audio") return "tts";
  if (s === "stt" || s === "whisper" || s === "transcribe" || s === "transcription") return "asr";
  if (s === "bgm" || s === "soundtrack" || s === "score") return "music";
  if (s === "matte" || s === "cutout" || s === "抠像") return "matting";
  if (s === "pexels" || s === "pixabay" || s === "unsplash" || s === "giphy" || s === "footage") return "stock";
  if (s === "llm" || s === "text") return "chat";
  return s;
}

/** 把设置字符串收成对象，失败就当空。raw 为假时走另一支，传给 JSON.parse。 */
function parseSettings(raw?: string): Record<string, any> {
  if (!raw) return {};
  try {
    const v = JSON.parse(raw);
    return v && typeof v === "object" && !Array.isArray(v) ? v : {};
  } catch {
    return {};
  }
}

/** 把草稿里的存储字段收成要保存的字符串。d 读取 bucket、region、cdn_base_url、service_type、path_style。 */
function settingsJSON(d: Draft): string {
  const st: Record<string, unknown> = {};
  if (d.bucket) st.bucket = d.bucket;
  if (d.region) st.region = d.region;
  if (d.cdn_base_url) st.cdn_base_url = d.cdn_base_url;
  if (d.service_type === "storage") st.path_style = !!d.path_style;
  return JSON.stringify(st);
}

/** 把模型字符串收成列表。raw 为假时用空字符串，去掉首尾空白后，按 , 切开，看开头，做映射，为假时走另一支，传给 JSON.parse。 */
function parseModels(raw: string): string[] {
  const s = (raw || "").trim();
  if (!s) return [];
  if (s.startsWith("[")) {
    try {
      const v = JSON.parse(s);
      if (Array.isArray(v)) return v.map(String).filter(Boolean);
    } catch {
      /* 解析失败就按逗号拆 */
    }
  }
  return s.split(",").map((x) => x.trim()).filter(Boolean);
}

/** 取出地址里的主机。url 为假时用空字符串，去掉首尾空白后，看是否包含，做替换，为假时走另一支，作为分支结果，作为后备值，拼进字符串。 */
function hostOf(url: string): string {
  const raw = (url || "").trim();
  if (!raw) return "";
  try {
    return new URL(raw.includes("://") ? raw : `https://${raw}`).host;
  } catch {
    return raw.replace(/^https?:\/\//, "").split("/")[0] || raw;
  }
}

/** 取出主机，路径不是根时再带上路径。url 为假时用空字符串，去掉首尾空白后，看是否包含，为假时走另一支，作为分支结果，拼进字符串，传给 hostOf。 */
function hostPath(url: string): string {
  const raw = (url || "").trim();
  if (!raw) return "";
  try {
    const u = new URL(raw.includes("://") ? raw : `https://${raw}`);
    const path = u.pathname.replace(/\/$/, "");
    return path && path !== "/" ? `${u.host}${path}` : u.host;
  } catch {
    return hostOf(raw);
  }
}

/** 给一条模板一个稳定的键。t 读取 provider、service_type、name、base_url、model。 */
function templateKey(t: MediaProvider): string {
  return [t.provider, t.service_type, t.name, t.base_url, t.model].join(":");
}

/** 按服务种类取出地址提示。kind 与 music、matting、stock、asr 比较是否相等。v 读取 musicUrlHint、mattingUrlHint、stockUrlHint、asrUrlHint、adapterUrlHint。 */
function adapterUrlHint(kind: Kind, v: ReturnType<typeof useCopy>["video"]): string | undefined {
  if (kind === "music") return v.musicUrlHint;
  if (kind === "matting") return v.mattingUrlHint;
  if (kind === "stock") return v.stockUrlHint;
  if (kind === "asr") return v.asrUrlHint;
  return v.adapterUrlHint;
}

/** 把厂商编号换成显示名，没有就留编号。id 用作下标，作为后备值。 */
function vendorLabel(id: string): string {
  return VENDOR_LABEL[id] || id || "Custom";
}

/** 造一份这种服务的空草稿。kind 写入字段 service_type。 */
function emptyDraft(kind: Kind): Draft {
  return {
    service_type: kind,
    provider: "custom",
    name: "",
    base_url: "",
    model: "",
    models: "",
    api_key: "",
    is_default: false,
    is_active: true,
  };
}

/** 把已保存的服务商收成草稿。p 读取 settings、id、service_type、provider、name、base_url 等。 */
function fromRow(p: MediaProvider): Draft {
  const st = parseSettings(p.settings);
  return {
    id: p.id,
    service_type: (p.service_type as Kind) || "image",
    provider: p.provider,
    name: p.name,
    base_url: p.base_url,
    model: p.model,
    models: parseModels(p.models).join(", "),
    api_key: "",
    is_default: !!p.is_default,
    is_active: p.is_active !== false,
    has_key: !!p.has_key,
    bucket: String(st.bucket || ""),
    region: String(st.region || ""),
    path_style: st.path_style !== false,
    cdn_base_url: String(st.cdn_base_url || ""),
  };
}

/** 把一条模板收成草稿。kind 传给 emptyDraft。t 读取 settings、provider、name、base_url、model、models。 */
function fromTemplate(kind: Kind, t: MediaProvider): Draft {
  const st = parseSettings(t.settings);
  return {
    ...emptyDraft(kind),
    provider: t.provider,
    name: t.name,
    base_url: t.base_url,
    model: t.model,
    models: parseModels(t.models).join(", "),
    bucket: String(st.bucket || ""),
    region: String(st.region || ""),
    path_style: st.path_style !== false,
    cdn_base_url: String(st.cdn_base_url || ""),
  };
}

/** 画一带说明的字段。label 用作标签，作为子节点渲染。hint 为真时才继续，作为子节点渲染。children 作为子节点渲染。 */
function Field({ label, hint, children }: { label: string; hint?: string; children: ReactNode }) {
  return (
    <label className="flex w-full flex-col gap-[var(--space-item)] text-[13px] font-medium text-foreground">
      <span>
        {label}
        {hint ? <span className="mt-0.5 block text-[12px] font-normal leading-[1.45] text-muted">{hint}</span> : null}
      </span>
      {children}
    </label>
  );
}

/** 画成功、进行或失败的状态。ok 为真时才继续。busy 为真时才继续。label 作为子节点渲染。 */
function StatusPill({ ok, busy, label }: { ok: boolean; busy?: boolean; label: string }) {
  return (
    <span
      className={cn(
        "inline-flex items-center gap-1.5 rounded-full px-2 py-0.5 text-[12px] font-medium",
        ok ? "bg-success/15 text-success" : busy ? "bg-accent/15 text-accent" : "bg-danger/10 text-danger",
      )}
    >
      <span className={cn("size-1.5 rounded-full", ok ? "bg-success" : busy ? "bg-accent" : "bg-danger")} aria-hidden />
      {label}
    </span>
  );
}

/** 安装还在解析、下载、解压或校验时为真。phase 是 resolving、downloading、extracting 或 verifying 时返回真。 */
function ffmpegBusy(phase?: string) {
  return phase === "resolving" || phase === "downloading" || phase === "extracting" || phase === "verifying";
}

/** 安装还在准备、安装或校验时为真。phase 是 preparing、installing 或 verifying 时返回真。 */
function remotionBusyPhase(phase?: string) {
  return phase === "preparing" || phase === "installing" || phase === "verifying";
}

type FFmpegInstall = {
  ready?: boolean;
  phase?: string;
  percent?: number;
  error?: string;
  archive?: string;
};

/** GenerationSettings 管工作室、服务商和画布通道。 */
export function GenerationSettings() {
  const copy = useCopy();
  const v = copy.video;
  const [templates, setTemplates] = useState<MediaProvider[]>([]);
  const [providers, setProviders] = useState<MediaProvider[]>([]);
  const [lang, setLang] = useState("zh");
  const [aspect, setAspect] = useState("9:16");
  const [platform, setPlatform] = useState("douyin");
  const [pubPlatforms, setPubPlatforms] = useState<{ id: string; label?: string }[]>([]);
  const [remotion, setRemotion] = useState<{
    ready?: boolean;
    hint?: string;
    node?: string;
    install?: FFmpegInstall;
  } | null>(null);
  const [remotionBusy, setRemotionBusy] = useState(false);
  const [ffmpeg, setFfmpeg] = useState(true);
  const [ffmpegInstall, setFfmpegInstall] = useState<FFmpegInstall>({});
  const [whisper, setWhisper] = useState(false);
  const [ytdlp, setYtdlp] = useState(false);
  const [ytdlpBusy, setYtdlpBusy] = useState(false);
  const [stamp, setStamp] = useState(false);
  const [regionTool, setRegionTool] = useState(false);
  const [browserCap, setBrowserCap] = useState(false);
  const [tick, setTick] = useState(0);

  /** 让列表再拉一次。 */
  function refresh() {
    setTick((n) => n + 1);
  }

  /** 把本机工具的状态写进界面。s 读取 ffmpeg、ffmpeg_install、whisper、ytdlp、stamp、region 等，为空时不读。 */
  function applyFFmpegStatus(s: any) {
    setFfmpeg(!!s?.ffmpeg);
    setFfmpegInstall(s?.ffmpeg_install && typeof s.ffmpeg_install === "object" ? s.ffmpeg_install : {});
    setWhisper(!!s?.whisper);
    setYtdlp(!!s?.ytdlp);
    setStamp(!!s?.stamp);
    setRegionTool(!!s?.region);
    setBrowserCap(!!s?.browser);
    const rem = s?.remotion && typeof s.remotion === "object" ? s.remotion : null;
    if (rem) setRemotion(rem);
  }

  useEffect(() => {
    void api.video.templates().then((t) => setTemplates(Array.isArray(t) ? t : [])).catch(() => {});
    void api.video.providers().then((p) => setProviders(Array.isArray(p) ? p : [])).catch(() => {});
    void api.video.settings().then((s) => {
      setLang(s?.content_language || "zh");
      const nextAspect = String(s?.studio_aspect || "9:16");
      setAspect(nextAspect === "16:9" || nextAspect === "1:1" ? nextAspect : "9:16");
      setPlatform(String(s?.studio_publish_platform || "douyin") || "douyin");
    }).catch(() => {});
    void api.video.status().then(applyFFmpegStatus).catch(() => {});
    void api.video.publishPlatforms().then((rows) => setPubPlatforms(Array.isArray(rows) ? rows : [])).catch(() => {});
    void api.video.remotionStatus().then((st) => setRemotion(st && typeof st === "object" ? st : null)).catch(() => setRemotion(null));
  }, [tick]);

  const ffmpegPhase = String(ffmpegInstall.phase || "");
  const ffmpegWorking = ffmpegBusy(ffmpegPhase);
  const remotionInstall = remotion?.install && typeof remotion.install === "object" ? remotion.install : {};
  const remotionPhase = String(remotionInstall.phase || "");
  const remotionWorking = remotionBusyPhase(remotionPhase);

  useEffect(() => {
    if (!ffmpegWorking && !remotionWorking) return;
    const t = window.setInterval(() => {
      void api.video.status().then(applyFFmpegStatus).catch(() => {});
    }, 400);
    return () => window.clearInterval(t);
  }, [ffmpegWorking, remotionWorking]);

  /** 安装 FFmpeg。 */
  async function installFFmpeg() {
    try {
      const s = await api.video.installFFmpeg();
      applyFFmpegStatus(s);
    } catch (e) {
      toast.error(api.errMessage(e));
    }
  }

  /** 安装 yt-dlp。 */
  async function installYTDLP() {
    setYtdlpBusy(true);
    try {
      await api.video.installYTDLP();
      setYtdlp(true);
    } catch (e) {
      toast.error(api.errMessage(e));
    } finally {
      setYtdlpBusy(false);
    }
  }

  const langChoice = lang === "zh" || lang === "zh-CN" ? "zh" : lang === "en" || lang === "en-US" ? "en" : "other";

  /** 保存内容语言。next 传给 setLang，传给 api.video.setSetting。 */
  async function setLanguage(next: string) {
    setLang(next);
    try {
      await api.video.setSetting("content_language", next);
    } catch (e) {
      toast.error(api.errMessage(e));
    }
  }

  /** 保存一项工作室设置。key 传给 api.video.setSetting。value 传给 api.video.setSetting。 */
  async function patchStudioSetting(key: string, value: string) {
    try {
      await api.video.setSetting(key, value);
    } catch (e) {
      toast.error(api.errMessage(e));
    }
  }

  /** 再查一次 Remotion 的状态。 */
  async function detectRemotion() {
    setRemotionBusy(true);
    try {
      const st = await api.video.remotionStatus();
      setRemotion(st && typeof st === "object" ? st : null);
    } catch (e) {
      toast.error(api.errMessage(e));
    } finally {
      setRemotionBusy(false);
    }
  }

  /** 安装 Remotion。 */
  async function installRemotion() {
    setRemotionBusy(true);
    try {
      const st = await api.video.installRemotion();
      setRemotion(st && typeof st === "object" ? st : null);
      const inst = st?.install && typeof st.install === "object" ? st.install : {};
      if (inst.phase === "error" && inst.error) toast.error(String(inst.error));
    } catch (e) {
      toast.error(api.errMessage(e));
    } finally {
      setRemotionBusy(false);
    }
  }

  return (
    <>
      <SettingSection
        id="generation-studio"
        title={copy.settings.sections.generationStudio}
        footnote={v.workshopFootnote}
        tight
      >
        <SettingRow title={v.language}>
          <SettingSegmented
            id="content-language"
            ariaLabel={v.language}
            value={langChoice}
            options={[
              { value: "zh", label: v.langZh },
              { value: "en", label: v.langEn },
              { value: "other", label: v.langOther },
            ]}
            onChange={(next) => {
              if (next === "other") return;
              void setLanguage(next);
            }}
          />
        </SettingRow>
        {langChoice === "other" ? (
          <SettingRow title={v.langOther} description={v.langOtherHint}>
            <Input
              className={CONTROL_LG}
              value={lang}
              aria-label={v.language}
              placeholder={v.langOtherHint}
              onChange={(e) => setLang(e.target.value)}
              onBlur={() => {
                const next = lang.trim();
                if (next) void setLanguage(next);
              }}
            />
          </SettingRow>
        ) : null}
        <SettingRow title={v.ffmpeg} description={v.ffmpegHint} list>
          <StatusPill
            ok={ffmpeg}
            busy={ffmpegWorking}
            label={
              ffmpeg
                ? v.ffmpegReady
                : ffmpegWorking
                  ? `${v.ffmpegInstalling} ${Math.max(0, Math.min(100, Math.round(Number(ffmpegInstall.percent) || 0)))}%`
                  : ffmpegPhase === "error"
                    ? v.ffmpegInstallFailed
                    : v.ffmpegMissing
            }
          />
          {!ffmpeg && !ffmpegWorking ? (
            <Button size="sm" variant="lift" onClick={() => void installFFmpeg()}>
              <Download aria-hidden />
              {v.ffmpegInstall}
            </Button>
          ) : null}
        </SettingRow>
        {ffmpegWorking ? <ProgressHairline value={Number(ffmpegInstall.percent) || 0} /> : null}
        {ffmpegPhase === "error" && !ffmpeg && ffmpegInstall.error ? (
          <p className="pb-2 text-[12px] leading-[1.45] text-danger">{String(ffmpegInstall.error)}</p>
        ) : null}
        <SettingRow title={v.studioAspect} description={v.studioAspectHint}>
          <SettingSegmented
            id="studio-aspect"
            ariaLabel={v.studioAspect}
            value={aspect}
            options={[
              { value: "9:16", label: "9:16" },
              { value: "16:9", label: "16:9" },
              { value: "1:1", label: "1:1" },
            ]}
            onChange={(next) => {
              setAspect(next);
              void patchStudioSetting("studio_aspect", next);
            }}
          />
        </SettingRow>
        <SettingRow title={v.studioPlatform} description={v.studioPlatformHint}>
          <Select
            value={platform}
            onValueChange={(next) => {
              setPlatform(next);
              void patchStudioSetting("studio_publish_platform", next);
            }}
          >
            <SelectTrigger className={CONTROL_LG} aria-label={v.studioPlatform}>
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              {(pubPlatforms.length
                ? pubPlatforms
                : [
                    { id: "douyin", label: "抖音" },
                    { id: "xhs", label: "小红书" },
                    { id: "bilibili", label: "B站" },
                  ]
              ).map((p) => (
                <SelectItem key={p.id} value={p.id}>
                  {p.label || p.id}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </SettingRow>
        <SettingRow title={v.remotionDetect} description={remotion?.ready ? v.remotionUseHint : remotion && !remotion.node ? v.remotionNeedNode : v.remotionDetectHint} border={false} list>
          <StatusPill
            ok={!!remotion?.ready}
            busy={remotionBusy || remotionWorking}
            label={
              remotion?.ready
                ? v.studio.remotionReady
                : remotionWorking
                  ? `${v.remotionInstalling} ${Math.max(0, Math.min(100, Math.round(Number(remotionInstall.percent) || 0)))}%`
                  : remotionPhase === "error"
                    ? v.remotionInstallFailed
                    : v.studio.remotionMissing
            }
          />
          {remotion?.ready ? (
            <Button size="sm" variant="ghost" disabled={remotionBusy} onClick={() => void detectRemotion()}>
              {v.remotionDetectNow}
            </Button>
          ) : (
            <Button size="sm" variant="lift" disabled={remotionBusy || remotionWorking || !remotion || !remotion.node} onClick={() => void installRemotion()} data-testid="remotion-install">
              <Download aria-hidden />
              {v.remotionInstall}
            </Button>
          )}
        </SettingRow>
        {remotionWorking ? <ProgressHairline value={Number(remotionInstall.percent) || 0} /> : null}
        {remotionPhase === "error" && !remotion?.ready && remotionInstall.error ? (
          <p className="pb-2 text-[12px] leading-[1.45] text-danger">{String(remotionInstall.error)}</p>
        ) : null}
      </SettingSection>

      <AdapterSection
        id="generation-image"
        title={copy.settings.sections.generationImage}
        kind="image"
        rows={providers}
        templates={templates}
        copy={copy}
        onChanged={refresh}
      />
      <AdapterSection
        id="generation-video"
        title={copy.settings.sections.generationVideo}
        kind="video"
        rows={providers}
        templates={templates}
        copy={copy}
        onChanged={refresh}
      />
      <AdapterSection
        id="generation-speech"
        title={copy.settings.sections.generationSpeech}
        kind="tts"
        rows={providers}
        templates={templates}
        copy={copy}
        onChanged={refresh}
        footnote={v.speechHint}
      />
      <AdapterSection
        id="generation-matting"
        title={copy.settings.sections.generationMatting}
        kind="matting"
        rows={providers}
        templates={templates}
        copy={copy}
        onChanged={refresh}
        description={v.mattingLead}
        footnote={v.mattingHint}
      />
      <AdapterSection
        id="generation-music"
        title={copy.settings.sections.generationMusic}
        kind="music"
        rows={providers}
        templates={templates}
        copy={copy}
        onChanged={refresh}
        description={v.musicLead}
        footnote={v.musicHint}
      />
      <AdapterSection
        id="generation-stock"
        title={copy.settings.sections.generationStock}
        kind="stock"
        rows={providers}
        templates={templates}
        copy={copy}
        onChanged={refresh}
        description={v.stockLead}
        footnote={v.stockHint}
      />
      <AdapterSection
        id="generation-asr"
        title={copy.settings.sections.generationAsr}
        kind="asr"
        rows={providers}
        templates={templates}
        copy={copy}
        onChanged={refresh}
        description={v.asrLead}
        footnote={v.asrHint}
        lead={
          <div data-testid="generation-whisper">
            <SettingRow title={v.whisper} description={v.whisperHint} list>
              <StatusPill
                ok={whisper}
                label={whisper ? v.whisperReady : v.whisperMissing}
              />
            </SettingRow>
            <details className="creation-more">
              <summary>{copy.video.film.toolsMore}</summary>
            <SettingRow title={copy.video.film.stamp} description={copy.video.film.stampHint} list>
              <StatusPill ok={stamp} label={stamp ? copy.video.film.stampReady : copy.video.film.stampMissing} />
            </SettingRow>
            <SettingRow title={copy.video.film.ytdlp} description={copy.video.film.ytdlpHint} list>
              <StatusPill ok={ytdlp} label={ytdlp ? copy.video.film.ytdlpReady : copy.video.film.ytdlpMissing} />
              {ytdlp ? null : (
                <Button size="sm" variant="lift" disabled={ytdlpBusy} onClick={() => void installYTDLP()}>
                  {ytdlpBusy ? copy.video.film.ytdlpInstalling : copy.video.film.ytdlpInstall}
                </Button>
              )}
            </SettingRow>
            <SettingRow title={copy.video.film.region} description={copy.video.film.regionHint} list>
              <StatusPill ok={regionTool} label={regionTool ? copy.video.film.regionReady : copy.video.film.regionMissing} />
            </SettingRow>
            <SettingRow title={copy.video.film.browser} description={copy.video.film.browserHint} list>
              <StatusPill ok={browserCap} label={browserCap ? copy.video.film.browserReady : copy.video.film.browserMissing} />
            </SettingRow>
            </details>
          </div>
        }
      />
      <AdapterSection
        id="generation-storage"
        title={copy.settings.sections.generationStorage}
        kind="storage"
        rows={providers}
        templates={templates}
        copy={copy}
        onChanged={refresh}
        footnote={v.storageHint}
      />
      <AdapterSection
        id="generation-search"
        title={copy.settings.sections.generationSearch}
        kind="search"
        rows={providers}
        templates={templates}
        copy={copy}
        onChanged={refresh}
        footnote={v.searchHint}
      />
      <AdapterSection
        id="generation-workflow"
        title={copy.settings.sections.generationWorkflow}
        kind="workflow"
        rows={providers}
        templates={templates}
        copy={copy}
        onChanged={refresh}
        footnote={v.workflowHint}
      />

      <CanvasChannelsSection copy={copy} />
    </>
  );
}

/** 编辑某一种服务的适配器列表。props 读取 copy、rows、kind、templates、id、title 等，调用 onChanged。 */
function AdapterSection(props: {
  id: string;
  title: string;
  kind: Kind;
  rows: MediaProvider[];
  templates: MediaProvider[];
  copy: ReturnType<typeof useCopy>;
  onChanged: () => void;
  description?: string;
  footnote?: string;
  lead?: ReactNode;
}) {
  const v = props.copy.video;
  const rows = useMemo(
    () => props.rows.filter((p) => serviceKind(p) === props.kind),
    [props.rows, props.kind],
  );
  const kindTemplates = useMemo(
    () => props.templates.filter((t) => serviceKind(t) === props.kind),
    [props.templates, props.kind],
  );
  const [draft, setDraft] = useState<Draft | null>(null);
  const [picking, setPicking] = useState(false);
  const [busy, setBusy] = useState("");
  const [pending, setPending] = useState<MediaProvider | null>(null);

  /** 保存一条服务商。d 读取 models、id、service_type、provider、name、base_url 等，传给 settingsJSON。 */
  async function save(d: Draft) {
    setBusy("save");
    try {
      const models = parseModels(d.models);
      await api.video.upsertProvider(
        {
          id: d.id,
          service_type: d.service_type,
          provider: d.provider || "custom",
          name: d.name || d.provider,
          base_url: d.base_url,
          model: d.model,
          models: JSON.stringify(models.length ? models : (d.model ? [d.model] : [])),
          settings: settingsJSON(d),
          is_default: d.is_default,
          is_active: d.is_active,
        },
        d.api_key,
      );
      setDraft(null);
      setPicking(false);
      props.onChanged();
      toast.success(props.copy.app.controlSaved);
    } catch (e) {
      toast.error(api.errMessage(e));
    } finally {
      setBusy("");
    }
  }

  /** 改一条服务商的启用、默认或模型。p 读取 model、models、id、service_type、provider、name 等。extra 读取 model、is_default、is_active。 */
  async function patchRow(p: MediaProvider, extra: { is_active?: boolean; is_default?: boolean; model?: string }) {
    try {
      const model = extra.model ?? p.model;
      const models = parseModels(p.models);
      if (model && !models.includes(model)) models.unshift(model);
      await api.video.upsertProvider(
        {
          id: p.id,
          service_type: p.service_type,
          provider: p.provider,
          name: p.name,
          base_url: p.base_url,
          model,
          models: JSON.stringify(models.length ? models : (model ? [model] : [])),
          is_default: extra.is_default ?? !!p.is_default,
          is_active: extra.is_active ?? p.is_active !== false,
          priority: p.priority,
        },
        "",
      );
      props.onChanged();
    } catch (e) {
      toast.error(api.errMessage(e));
    }
  }

  /** 探测一条服务商。id 加上 test:，传给 api.video.testProvider。 */
  async function test(id: string) {
    setBusy("test:" + id);
    try {
      const r = await api.video.testProvider(id);
      if (r?.ok) toast.success(props.copy.settings.testOk);
      else toast.error(String(r?.error || props.copy.settings.testFail));
    } catch (e) {
      toast.error(api.errMessage(e));
    } finally {
      setBusy("");
    }
  }

  const showPicker = picking || (rows.length === 0 && !draft);

  return (
    <SettingSection id={props.id} title={props.title} description={props.description} footnote={props.footnote || v.multiAdapterHint} tight>
      {props.lead}
      {rows.length === 0 && !draft ? <SettingEmpty compact>{v.noAdapter}</SettingEmpty> : null}

      {rows.map((p) => {
        const open = draft?.id === p.id;
        const models = parseModels(p.models);
        if (p.model && !models.includes(p.model)) models.unshift(p.model);
        return (
          <div key={p.id} className={cn("rounded-xl px-1", open && "bg-background/40")}>
            <div className="flex min-w-0 flex-wrap items-center gap-x-3 gap-y-1.5 py-1">
              <ProviderMark id={p.provider} className="size-8" />
              <button
                type="button"
                className="min-w-0 flex-1 basis-[12rem] text-left"
                onClick={() => {
                  setPicking(false);
                  setDraft(open ? null : fromRow(p));
                }}
              >
                <div className="truncate text-[13px] font-medium leading-[1.4] text-foreground">{p.name}</div>
                <div className="mt-0.5 truncate font-mono text-[11px] leading-[1.4] text-muted">
                  {[vendorLabel(p.provider), hostPath(p.base_url)].filter(Boolean).join(" · ")}
                </div>
              </button>
              <div className="ml-auto flex shrink-0 flex-wrap items-center justify-end gap-1.5">
                {p.is_default ? (
                  <span className="rounded-full bg-lift px-2 py-0.5 text-[11px] font-medium text-foreground">{v.makeDefault}</span>
                ) : (
                  <button
                    type="button"
                    className="rounded-full px-2 py-0.5 text-[11px] font-medium text-muted hover:bg-lift hover:text-foreground"
                    onClick={() => void patchRow(p, { is_default: true })}
                  >
                    {v.setDefault}
                  </button>
                )}
                <span
                  className={cn(
                    "rounded-full px-2 py-0.5 text-[11px] font-medium",
                    p.provider === "cas" || p.has_key ? "bg-success/15 text-success" : "bg-danger/10 text-danger",
                  )}
                >
                  {p.provider === "cas" || p.has_key ? v.keySaved : v.noKey}
                </span>
                <div
                  onClick={(e) => e.stopPropagation()}
                  onPointerDown={(e) => e.stopPropagation()}
                >
                  <Switch
                    checked={p.is_active !== false}
                    aria-label={v.enabled}
                    onCheckedChange={(on) => void patchRow(p, { is_active: on })}
                  />
                </div>
              </div>
            </div>
            {models.length ? (
                <div className="flex flex-wrap gap-1 pb-1.5 pl-11" role="group" aria-label={v.defaultModel}>
                  {models.map((m) => {
                    const on = m === p.model;
                    return (
                      <button
                        key={m}
                        type="button"
                        title={v.defaultModel}
                        aria-pressed={on}
                        className={cn(
                          "max-w-[14rem] truncate rounded-full px-2 py-0.5 font-mono text-[11px]",
                          on ? "bg-lift text-foreground" : "text-muted hover:bg-lift/70 hover:text-foreground",
                        )}
                        onClick={() => void patchRow(p, { model: m })}
                      >
                        {m}
                      </button>
                    );
                  })}
                </div>
            ) : null}
            {open && draft ? (
              <AdapterForm
                draft={draft}
                setDraft={setDraft}
                copy={props.copy}
                busy={busy}
                onSave={() => void save(draft)}
                onTest={() => void test(p.id)}
                onDelete={p.id === "cas-local" ? undefined : () => setPending(p)}
              />
            ) : null}
          </div>
        );
      })}

      {draft && !draft.id ? (
        <AdapterForm
          draft={draft}
          setDraft={(d) => {
            setDraft(d);
            if (!d) setPicking(false);
          }}
          copy={props.copy}
          busy={busy}
          onSave={() => void save(draft)}
        />
      ) : null}

      {showPicker && !draft ? (
        <div className="flex flex-col gap-0.5">
          <p className="px-1 text-[12px] leading-[1.4] text-muted">{v.templateHint}</p>
          {kindTemplates.map((t) => (
            <AdapterPickRow
              key={templateKey(t)}
              testId={`adapter-template-${(t.name || t.provider).toLowerCase().replace(/[^a-z0-9]+/g, "-")}`}
              mark={t.provider}
              title={t.name}
              subtitle={[vendorLabel(t.provider), t.model || hostPath(t.base_url) || v.pasteGateway].filter(Boolean).join(" · ")}
              onClick={() => {
                setDraft(fromTemplate(props.kind, t));
                setPicking(false);
              }}
            />
          ))}
          <AdapterPickRow
            testId="adapter-custom"
            mark="custom"
            title={v.customProvider}
            subtitle={v.customProviderHint}
            trailing={<Plus className="size-3.5 text-muted" aria-hidden />}
            onClick={() => {
              setDraft(emptyDraft(props.kind));
              setPicking(false);
            }}
          />
        </div>
      ) : rows.length > 0 && !draft ? (
        <button
          type="button"
          className="flex min-h-9 w-full items-center gap-3 rounded-md px-1 text-left text-muted transition-colors hover:bg-lift/70 hover:text-foreground"
          onClick={() => setPicking(true)}
        >
          <span className="grid size-8 shrink-0 place-items-center rounded-[22%] bg-lift text-muted">
            <Plus className="size-3.5" aria-hidden />
          </span>
          <span className="text-[13px] font-medium">{v.addAdapter}</span>
        </button>
      ) : null}

      <ConfirmDialog
        open={!!pending}
        title={v.delete}
        body={v.deleteAdapter}
        danger
        confirmLabel={v.delete}
        onCancel={() => setPending(null)}
        onConfirm={async () => {
          if (!pending) return;
          await api.video.deleteProvider(pending.id);
          if (draft?.id === pending.id) setDraft(null);
          setPending(null);
          props.onChanged();
        }}
      />
    </SettingSection>
  );
}

/**
 * 画一行可选的适配器。
 * mark 写到 ProviderMark 的 id。
 * title 作为子节点渲染。
 * subtitle 为真时才继续，作为子节点渲染。
 * onClick 写到 button 的 onClick。
 * testId 写到 button 的 data-testid。
 * trailing 为真时才继续，作为子节点渲染。
 */
function AdapterPickRow({
  mark,
  title,
  subtitle,
  onClick,
  testId,
  trailing,
}: {
  mark: string;
  title: string;
  subtitle?: string;
  onClick: () => void;
  testId?: string;
  trailing?: ReactNode;
}) {
  return (
    <button
      type="button"
      data-testid={testId}
      className="flex min-h-10 w-full items-center gap-3 rounded-md px-1 text-left transition-colors hover:bg-lift/70"
      onClick={onClick}
    >
      <ProviderMark id={mark} className="size-8" />
      <span className="min-w-0 flex-1">
        <span className="block text-[13px] font-medium leading-[1.4] text-foreground">{title}</span>
        {subtitle ? (
          <span className="mt-0.5 block truncate font-mono text-[12px] leading-[1.4] text-muted">{subtitle}</span>
        ) : null}
      </span>
      {trailing ? <span className="flex shrink-0 text-muted">{trailing}</span> : null}
    </button>
  );
}

/**
 * 画适配器的表单和保存。
 * draft 读取 models、name、provider、base_url、service_type、model 等，展开传递。
 * setDraft 被调用，并传入 { ...draft, ...p }。
 * copy 读取 video、settings。
 * busy 为假时走另一支。
 * onSave 写到 Button 的 onClick。
 * onTest 为真时才继续，写到 Button 的 onClick。
 * onDelete 为真时才继续，写到 Button 的 onClick。
 */
function AdapterForm({
  draft,
  setDraft,
  copy,
  busy,
  onSave,
  onTest,
  onDelete,
}: {
  draft: Draft;
  setDraft: (d: Draft | null) => void;
  copy: ReturnType<typeof useCopy>;
  busy: string;
  onSave: () => void;
  onTest?: () => void;
  onDelete?: () => void;
}) {
  const v = copy.video;
  const models = parseModels(draft.models);
  /** 把局部字段写回当前草稿。 */
  /** patch 把局部字段写进当前草稿。p 展开传递。 */
  const patch = (p: Partial<Draft>) => setDraft({ ...draft, ...p });
  const canSave = !!draft.name.trim() && (draft.provider === "cas" || !!draft.base_url.trim());
  return (
    <div className="flex flex-col gap-[var(--space-group)] border-t border-border/60 px-1 py-3">
      <Field label={v.adapterName}>
        <Input className={CONTROL_LG} value={draft.name} onChange={(e) => patch({ name: e.target.value })} />
      </Field>
      <Field label={v.vendor}>
        <Input className={cn(CONTROL_LG, "font-mono")} value={draft.provider} onChange={(e) => patch({ provider: e.target.value })} />
      </Field>
      <Field label={copy.settings.baseUrl} hint={!draft.base_url.trim() ? adapterUrlHint(draft.service_type, v) : undefined}>
        <Input className={cn(CONTROL_LG, "font-mono")} value={draft.base_url} placeholder={!draft.base_url.trim() ? v.pasteGateway : undefined} onChange={(e) => patch({ base_url: e.target.value })} />
      </Field>
      {usesModels(draft.service_type) ? (
        <>
          <Field label={copy.settings.model}>
            {models.length ? (
              <Select value={draft.model || models[0]} onValueChange={(value) => patch({ model: value })}>
                <SelectTrigger className="h-8 w-full font-mono">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  {models.map((m) => (
                    <SelectItem key={m} value={m}>{m}</SelectItem>
                  ))}
                </SelectContent>
              </Select>
            ) : (
              <Input className={cn(CONTROL_LG, "font-mono")} value={draft.model} onChange={(e) => patch({ model: e.target.value })} />
            )}
          </Field>
          <Field label={v.modelsCsv}>
            <Input className={cn(CONTROL_LG, "font-mono")} value={draft.models} onChange={(e) => patch({ models: e.target.value })} />
          </Field>
        </>
      ) : null}
      {draft.service_type === "storage" ? (
        <>
          <Field label={v.bucket}>
            <Input className={cn(CONTROL_LG, "font-mono")} value={draft.bucket || ""} onChange={(e) => patch({ bucket: e.target.value })} />
          </Field>
          <Field label={v.region}>
            <Input className={cn(CONTROL_LG, "font-mono")} value={draft.region || ""} onChange={(e) => patch({ region: e.target.value })} />
          </Field>
          <Field label={v.cdnBase}>
            <Input className={cn(CONTROL_LG, "font-mono")} value={draft.cdn_base_url || ""} onChange={(e) => patch({ cdn_base_url: e.target.value })} />
          </Field>
          <SettingRow title={v.pathStyle} list>
            <Switch checked={draft.path_style !== false} onCheckedChange={(on) => patch({ path_style: on })} />
          </SettingRow>
        </>
      ) : null}
      {draft.provider !== "cas" ? (
        <Field label={copy.settings.apiKey} hint={draft.has_key ? v.keySaved : copy.settings.apiKeyPh}>
          <Input
            className={cn(CONTROL_LG, "font-mono")}
            type="password"
            autoComplete="off"
            placeholder={draft.id && draft.has_key ? "••••••••" : copy.settings.apiKeyPh}
            value={draft.api_key}
            onChange={(e) => patch({ api_key: e.target.value })}
          />
        </Field>
      ) : null}
      <SettingRow title={v.enabled} list>
        <Switch checked={draft.is_active} onCheckedChange={(on) => patch({ is_active: on })} />
      </SettingRow>
      <SettingRow title={v.makeDefault} list>
        <Switch checked={draft.is_default} onCheckedChange={(on) => patch({ is_default: on })} />
      </SettingRow>
      <div className="flex flex-wrap items-center justify-end gap-2">
        {onDelete ? (
          <Button size="sm" variant="ghost" className="mr-auto text-danger" onClick={onDelete}>{v.delete}</Button>
        ) : null}
        <Button size="sm" variant="ghost" onClick={() => setDraft(null)}>{copy.settings.discard}</Button>
        {onTest && draft.id ? (
          <Button size="sm" variant="lift" disabled={!!busy} onClick={onTest}>{copy.settings.testConnection}</Button>
        ) : null}
        <Button size="sm" disabled={!!busy || !canSave} title={!canSave ? v.needBaseUrl : undefined} onClick={onSave}>
          {copy.settings.saveChanges}
        </Button>
      </div>
    </div>
  );
}

/** 取出响应里的 data，没有就用响应本身。env 读取 data，为空时不读，作为后备值。 */
function envelopeData(env: any) {
  return env?.data ?? env ?? {};
}

/** 编辑画布通道。props 读取 copy。 */
function CanvasChannelsSection(props: { copy: ReturnType<typeof useCopy> }) {
  const copy = props.copy;
  const [channels, setChannels] = useState<any[]>([]);
  const [plugins, setPlugins] = useState<any[]>([]);
  const [draft, setDraft] = useState({ pluginId: "", name: "", baseUrl: "", apiKey: "", capability: "image" });
  const [busy, setBusy] = useState(false);
  const [exposeMcp, setExposeMcp] = useState(false);

  /** 拉取通道、插件目录和是否暴露 MCP。 */
  async function load() {
    try {
      const ch = envelopeData(await api.video.canvasHttp("GET", "channels"));
      setChannels(Array.isArray(ch.channels) ? ch.channels : []);
    } catch {
      setChannels([]);
    }
    try {
      const cat = envelopeData(await api.video.canvasHttp("GET", "plugins/catalog"));
      const list = cat.providers || cat.plugins || cat.catalog || cat.items || [];
      setPlugins(Array.isArray(list) ? list : []);
    } catch {
      setPlugins([]);
    }
    try {
      const s = await api.video.settings();
      const v = String(s?.canvas_expose_mcp || "").toLowerCase();
      setExposeMcp(v === "1" || v === "true" || v === "on");
    } catch {
      setExposeMcp(false);
    }
  }

  useEffect(() => {
    void load();
  }, []);

  /** 保存一条画布通道。 */
  async function save() {
    setBusy(true);
    try {
      await api.video.canvasHttp("POST", "channels", {
        body: {
          pluginId: draft.pluginId,
          name: draft.name || draft.pluginId || "Channel",
          baseUrl: draft.baseUrl,
          capability: draft.capability,
        },
        api_key: draft.apiKey,
      });
      setDraft({ pluginId: "", name: "", baseUrl: "", apiKey: "", capability: "image" });
      await load();
      toast.success(copy.app.controlSaved);
    } catch (e) {
      toast.error(api.errMessage(e));
    } finally {
      setBusy(false);
    }
  }

  /** 探测一条画布通道。id 拼进字符串。 */
  async function test(id: string) {
    try {
      await api.video.canvasHttp("POST", `channels/${id}/test`);
      toast.success(copy.settings.testConnection);
    } catch (e) {
      toast.error(api.errMessage(e));
    }
  }

  const extras = channels.filter((ch) => !String(ch.id || "").startsWith("ch-"));

  return (
    <SettingSection id="generation-canvas" title={copy.settings.sections.generationCanvas} footnote={copy.video.canvasChannelsHint} tight>
      {extras.length === 0 ? (
        <SettingEmpty compact>{copy.video.canvasChannelsEmpty}</SettingEmpty>
      ) : (
        extras.map((ch) => (
          <SettingRow key={ch.id} title={ch.name || ch.pluginId || ch.id} list>
            <div className="flex min-w-0 flex-1 items-center justify-between gap-2">
              <span className="truncate text-[12px] text-muted">{ch.pluginId} · {ch.capability} · {ch.baseUrl}</span>
              <Button size="sm" variant="ghost" onClick={() => void test(ch.id)}>{copy.settings.testConnection}</Button>
            </div>
          </SettingRow>
        ))
      )}
      <SettingRow title={copy.video.canvasMcp} description={copy.video.canvasMcpHint}>
        <Switch
          checked={exposeMcp}
          onCheckedChange={(on) => {
            setExposeMcp(on);
            void api.video.setSetting("canvas_expose_mcp", on ? "1" : "0");
          }}
        />
      </SettingRow>
      <SettingRow title={copy.video.canvasPlugin}>
        <Select value={draft.pluginId || "__none__"} onValueChange={(id) => setDraft((d) => ({ ...d, pluginId: id === "__none__" ? "" : id }))}>
          <SelectTrigger className={CONTROL_LG} aria-label={copy.video.canvasPlugin}><SelectValue placeholder={copy.video.canvasPlugin} /></SelectTrigger>
          <SelectContent>
            <SelectItem value="__none__">—</SelectItem>
            {plugins.slice(0, 80).map((p) => {
              const id = String(p.id || p.pluginId || p.name || "");
              return id ? <SelectItem key={id} value={id}>{p.name || id}</SelectItem> : null;
            })}
          </SelectContent>
        </Select>
      </SettingRow>
      <SettingRow title={copy.video.canvasCapability}>
        <SettingSegmented
          id="canvas-channel-capability"
          ariaLabel={copy.video.canvasCapability}
          value={draft.capability}
          options={[
            { value: "image", label: copy.video.kindImage },
            { value: "video", label: copy.video.kindVideo },
            { value: "tts", label: copy.video.kindSpeech },
          ]}
          onChange={(next) => setDraft((d) => ({ ...d, capability: next }))}
        />
      </SettingRow>
      <SettingRow title={copy.video.canvasBaseUrl}>
        <Input className={CONTROL_LG} value={draft.baseUrl} onChange={(e) => setDraft((d) => ({ ...d, baseUrl: e.target.value }))} aria-label={copy.video.canvasBaseUrl} />
      </SettingRow>
      <SettingRow title={copy.video.canvasApiKey} border={false}>
        <div className="flex min-w-0 flex-1 items-center gap-2">
          <Input className={CONTROL_LG} type="password" value={draft.apiKey} onChange={(e) => setDraft((d) => ({ ...d, apiKey: e.target.value }))} aria-label={copy.video.canvasApiKey} />
          <Button size="sm" disabled={busy || (!draft.pluginId && !draft.baseUrl.trim())} onClick={() => void save()}>{copy.video.canvasSave}</Button>
        </div>
      </SettingRow>
    </SettingSection>
  );
}
