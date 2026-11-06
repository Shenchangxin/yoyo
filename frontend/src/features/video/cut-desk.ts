/**
 * 本文件拥有剪辑台的本地工程指针、片段和字幕的读取，以及就绪探测和 ffmpeg 安装进度。
 * 不拥有时间线界面。探测只读视频状态和语音提供方菜单；安装 ffmpeg 会让服务端下载组件。
 */
import { useEffect } from "react";
import { create } from "zustand";
import * as api from "../../lib/client";
import { videoItemOf } from "./video-item";

/** CutCue 是一条字幕，带起止、文本和是否略过。 */
export type CutCue = { start?: number; end?: number; text?: string; omit?: boolean };
/** CutClip 是一个候选片段，带哈希、起止、分数和地址。 */
export type CutClip = {
  id?: string;
  hash?: string;
  start?: number;
  end?: number;
  title?: string;
  scores?: Record<string, number>;
  media_url?: string;
};
/** CutProject 是剪辑工程在界面上用到的字段，含素材、成片、字幕和分轨地址。 */
export type CutProject = {
  id?: string;
  title?: string;
  session_id?: string;
  source_hash?: string;
  kind?: string;
  reference_hash?: string;
  note?: string;
  result_hash?: string;
  duration?: number;
  stale?: boolean;
  media_url?: string;
  source_url?: string;
  result_url?: string;
  poster_url?: string;
  dub_url?: string;
  voice_url?: string;
  music_url?: string;
  updated_at?: string;
  breakdown?: { scenes?: number[]; clips?: CutClip[] };
  edit_list?: { cuts?: CutClip[] };
  clips?: CutClip[];
  captions?: { cues?: CutCue[] };
  stems?: { dub?: string; voice?: string; music?: string };
};

type CutDeskState = {
  sessionId: string;
  project: CutProject | null;
  /** 用户离开剪辑台回到创作入口。不要把已绑定的成片再弹开。 */
  parked: boolean;
  /** 停靠前那条剪辑。首页用它做「继续剪」，重新打开后清空。 */
  resumeId: string;
  resumeTitle: string;
  setProject: (project: CutProject | null, sessionId?: string) => void;
  noteSession: (sessionId: string) => void;
  adoptServer: (sessionId: string, project: CutProject | null) => void;
  park: () => void;
  clear: () => void;
};

/** useCutDesk 记住当前剪辑工程、会话，以及用户是否已经离开这张桌子。 */
export const useCutDesk = create<CutDeskState>(/** set 写入工程、会话或停靠。 */ (set) => ({
  sessionId: "",
  project: null,
  parked: false,
  resumeId: "",
  resumeTitle: "",
  /** 写入工程并取消停靠；没给会话就沿用原来的。重新打开后不再提示继续剪。project 写进状态。sessionId 有传入时换上，否则留原来的。 */
  setProject: (project, sessionId) => set(/** s 没传入会话时留下原来的 sessionId。 */ (s) => ({
    project,
    parked: false,
    resumeId: "",
    resumeTitle: "",
    sessionId: sessionId ?? s.sessionId,
  })),
  /** 会话变了就清掉工程、停靠和「继续剪」。sessionId 和当前会话比较。 */
  noteSession: (sessionId) => set(/** s 会话没变就整份返回，变了就清掉工程并取消停靠。 */ (s) => (
    s.sessionId === sessionId ? s : { sessionId, project: null, parked: false, resumeId: "", resumeTitle: "" }
  )),
  /** 同一会话上，有素材哈希的工程会打开；停靠中则保持关掉；本地有素材而传来的没有时不覆盖。sessionId 和当前会话比较。project 看有没有 source_hash，决定打开、清掉或不覆盖。 */
  adoptServer: (sessionId, project) => set(/** s 比较会话、停靠和本地素材哈希，决定打开、保持关掉或不覆盖。 */ (s) => {
    if (sessionId && s.sessionId && s.sessionId !== sessionId) return s;
    if (s.parked && s.sessionId === sessionId) return { sessionId, project: null, parked: true };
    if (s.sessionId === sessionId && s.project?.source_hash && !project?.source_hash) return s;
    if (project?.source_hash) return { project, sessionId, parked: false };
    if (s.sessionId === sessionId) return { project: null, sessionId, parked: s.parked };
    return s;
  }),
  /** 清掉工程并标记停靠，会话 id 保留。有源片的工程记下编号和标题，首页可以继续剪。 */
  park: () => set(/** s 有源片就记下这条，否则留下上次记下的。 */ (s) => {
    const keep = !!(s.project?.id && s.project.source_hash);
    return {
      project: null,
      parked: true,
      sessionId: s.sessionId,
      resumeId: keep ? String(s.project?.id) : s.resumeId,
      resumeTitle: keep ? String(s.project?.title || "") : s.resumeTitle,
    };
  }),
  /** 工程、会话、停靠和继续剪一起清掉。 */
  clear: () => set({ project: null, sessionId: "", parked: false, resumeId: "", resumeTitle: "" }),
}));

/** cutClips 优先用工程上的 clips，否则用 breakdown 里的片段。project 没有就返回空数组，否则取 clips，再否则取 breakdown.clips。 */
export function cutClips(project: CutProject | null): CutClip[] {
  if (!project) return [];
  return project.clips || project.breakdown?.clips || [];
}

/** cutCues 取出字幕条目，没有就是空数组。project 取 captions.cues，没有就返回空数组。 */
export function cutCues(project: CutProject | null): CutCue[] {
  return project?.captions?.cues || [];
}

/** FFmpegInstall 是服务端回报的 ffmpeg 安装阶段、百分比和错误。 */
export type FFmpegInstall = { phase?: string; percent?: number; error?: string };

/** CutReadiness 记下 ffmpeg、whisper、语音提供方是否可用，ffmpeg 的安装进度，以及这次探测是否已经返回。 */
export type CutReadiness = { ffmpeg: boolean; whisper: boolean; speech: boolean; loaded: boolean; install: FFmpegInstall };

const INSTALL_PHASES = new Set(["resolving", "downloading", "extracting", "verifying"]);

/** ffmpegInstalling 在安装还停在解析、下载、解压或校验时为真。 */
export function ffmpegInstalling(install?: FFmpegInstall): boolean {
  return INSTALL_PHASES.has(String(install?.phase || ""));
}

/** 把状态里的安装信息收成对象，不是对象就当空。 */
function installOf(status: any): FFmpegInstall {
  const raw = status?.ffmpeg_install;
  return raw && typeof raw === "object" ? raw as FFmpegInstall : {};
}

type ReadinessStore = CutReadiness & {
  probe: () => Promise<void>;
  installFFmpeg: () => Promise<void>;
};

let probing: Promise<void> | null = null;
let pollTimer = 0;
let ffmpegListening = false;

/** 把安装快照写进就绪状态。已经不在安装就停掉兜底轮询，再探测一次语音和 whisper。install 是服务端快照。ffmpeg 为真表示本机已经能用。 */
function applyInstall(install: FFmpegInstall, ffmpeg?: boolean) {
  useReadinessStore.setState({ install, ...(ffmpeg == null ? {} : { ffmpeg }) });
  if (ffmpegInstalling(install)) return;
  if (pollTimer) {
    window.clearInterval(pollTimer);
    pollTimer = 0;
  }
  void useReadinessStore.getState().probe();
}

/** 听安装进度。桌面端推 ffmpeg 事件；不在桌面壳里就只靠兜底轮询。 */
function listenFfmpeg() {
  if (ffmpegListening || typeof window === "undefined") return;
  ffmpegListening = true;
  void import("@wailsio/runtime").then(/** mod 取出 Events，没有 On 就不订。 */ (mod) => {
    const Events = (mod as { Events?: { On?: (name: string, cb: (ev: unknown) => void) => () => void } }).Events;
    if (!Events?.On) return;
    Events.On("yoyo:item", /** ev 只认 ffmpeg 安装快照。 */ (ev) => {
      const { kind, payload } = videoItemOf(ev);
      if (kind !== "ffmpeg") return;
      applyInstall(payload as FFmpegInstall, payload.ready === true || payload.phase === "ready");
    });
  }).catch(() => {});
}

/** 安装还在进行时，进度靠事件推进；事件丢掉了就每 10 秒问一次。结束后再探测一次语音和 whisper。 */
function pollInstall() {
  if (pollTimer || typeof window === "undefined") return;
  listenFfmpeg();
  pollTimer = window.setInterval(() => {
    void api.video.status().then(/** status 读 ffmpeg 和安装快照。 */ (status) => {
      applyInstall(installOf(status), !!status?.ffmpeg);
    }).catch(() => {});
  }, 10_000);
}

const useReadinessStore = create<ReadinessStore>((set) => ({
  ffmpeg: false,
  whisper: false,
  speech: false,
  loaded: false,
  install: {},
  /** 读视频状态和语音提供方菜单。同时只跑一次。 */
  probe: () => {
    if (probing) return probing;
    probing = Promise.all([
      api.video.status().catch(() => ({})),
      api.video.providerMenu("speech").catch(() => []),
    ]).then(([status, speech]) => {
      const install = installOf(status);
      set({
        ffmpeg: !!status?.ffmpeg,
        whisper: !!status?.whisper,
        speech: (Array.isArray(speech) && speech.length > 0) || !!status?.edge_tts,
        loaded: true,
        install,
      });
      if (ffmpegInstalling(install)) pollInstall();
    }).finally(() => { probing = null; });
    return probing;
  },
  /** 开始下载 ffmpeg，并跟着进度轮询到结束。 */
  installFFmpeg: async () => {
    const out = await api.video.installFFmpeg();
    const install = installOf(out);
    set({ ffmpeg: !!out?.ffmpeg, install, loaded: true });
    if (ffmpegInstalling(install)) pollInstall();
    else void useReadinessStore.getState().probe();
  },
}));

/** useCutReadiness 读视频状态和语音提供方菜单。每次挂载都重探一次，装好 ffmpeg 或启用语音账号后回到这张桌子就能用。 */
export function useCutReadiness(): CutReadiness & { installFFmpeg: () => Promise<void> } {
  const ready = useReadinessStore();
  useEffect(() => {
    void ready.probe();
    // 只在挂载时探测。安装进度由 pollInstall 推进。
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);
  return ready;
}
