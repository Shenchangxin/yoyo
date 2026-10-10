import { useEffect, useRef, useState } from "react";
import { Mic, MicOff, PhoneOff, Volume2, VolumeX } from "lucide-react";
import { useCopy } from "../../lib/i18n";
import * as api from "../../lib/client";
import { useUI } from "../../lib/store";
import { cn } from "../../lib/utils";
import type { VoicePhase } from "./types";

const MAX_TURNS = 6;
const MAX_MS = 15 * 60 * 1000;

export function CallView(props: {
  sessionId: string;
  paused?: boolean;
  onHangup: () => void;
  lastAssistant?: string;
}) {
  const copy = useCopy();
  const setPhase = useUI((s) => s.setVoicePhase);
  const phase = useUI((s) => s.voicePhase);
  const [muteMic, setMuteMic] = useState(false);
  const [muteSpk, setMuteSpk] = useState(false);
  const [caption, setCaption] = useState("");
  const [setup, setSetup] = useState("");
  const [elapsed, setElapsed] = useState(0);
  const started = useRef(Date.now());
  const turns = useRef(0);
  const rec = useRef<MediaRecorder | null>(null);
  const stream = useRef<MediaStream | null>(null);
  const audio = useRef<HTMLAudioElement | null>(null);
  const transcript = useRef("");
  const hangupRef = useRef(props.onHangup);
  hangupRef.current = props.onHangup;
  const lastAsst = useRef(props.lastAssistant || "");
  lastAsst.current = props.lastAssistant || "";

  useEffect(() => {
    const t = window.setInterval(() => setElapsed(Date.now() - started.current), 1000);
    return () => window.clearInterval(t);
  }, []);

  useEffect(() => {
    if (props.paused || elapsed > MAX_MS || turns.current >= MAX_TURNS) {
      void endCall();
    }
  }, [props.paused, elapsed]);

  useEffect(() => {
    void api.setCompanionCall(true);
    void startLoop();
    return () => {
      void endCall(true);
    };
  }, []);

  async function startLoop() {
    try {
      stream.current = await navigator.mediaDevices.getUserMedia({ audio: true });
    } catch {
      setSetup(copy.voice.needMic);
      setPhase("off");
      return;
    }
    setPhase("listening");
    await captureUtterance();
  }

  async function captureUtterance() {
    if (!stream.current || muteMic) return;
    const chunks: Blob[] = [];
    const mr = new MediaRecorder(stream.current);
    rec.current = mr;
    mr.ondataavailable = (e) => { if (e.data.size) chunks.push(e.data); };
    const done = new Promise<void>((resolve) => { mr.onstop = () => resolve(); });
    mr.start();
    await new Promise((r) => setTimeout(r, 4000));
    if (mr.state === "recording") mr.stop();
    await done;
    const blob = new Blob(chunks, { type: mr.mimeType || "audio/webm" });
    const b64 = await blobToB64(blob);
    setPhase("thinking");
    try {
      const text = await api.voiceTranscribe("audio.webm", b64);
      if (!text.trim()) {
        if (turns.current < MAX_TURNS && !props.paused) void captureUtterance();
        return;
      }
      transcript.current += (transcript.current ? "\n" : "") + text;
      setCaption(text);
      await api.send(props.sessionId, text);
      turns.current += 1;
      setPhase("speaking");
      await speakLatest();
    } catch (e: any) {
      const msg = String(e?.message || e);
      if (/speech connection not configured|not configured/i.test(msg)) setSetup(copy.voice.needSetup);
      else setSetup(msg);
      setPhase("off");
      return;
    }
    if (turns.current < MAX_TURNS && !props.paused) {
      setPhase("listening");
      void captureUtterance();
    } else {
      void endCall();
    }
  }

  async function speakLatest() {
    for (let i = 0; i < 16 && lastAsst.current === caption; i++) {
      await new Promise((r) => setTimeout(r, 400));
    }
    const text = lastAsst.current || caption;
    if (!text.trim() || muteSpk) return;
    try {
      const { audio: a64, contentType } = await api.voiceSpeak(text);
      const url = `data:${contentType};base64,${a64}`;
      const el = new Audio(url);
      audio.current = el;
      await el.play().catch(() => {});
    } catch {
      /* keep captions */
    }
  }

  async function endCall(fromUnmount = false) {
    rec.current?.stop();
    stream.current?.getTracks().forEach((t) => t.stop());
    audio.current?.pause();
    setPhase("off" as VoicePhase);
    void api.setCompanionCall(false);
    const sec = Math.round((Date.now() - started.current) / 1000);
    if (props.sessionId) api.voiceReceipt(props.sessionId, sec, transcript.current);
    if (!fromUnmount) hangupRef.current();
  }

  const mm = String(Math.floor(elapsed / 60000)).padStart(2, "0");
  const ss = String(Math.floor((elapsed % 60000) / 1000)).padStart(2, "0");
  return (
    <div className="flex h-full flex-col items-center justify-between px-4 py-3" data-testid="call-view">
      <div className="text-[12px] text-muted">{mm}:{ss} · {copy.voice[phase] || phase}</div>
      {setup ? <div className="text-center text-[12px] text-warning">{setup}</div> : null}
      <p className="max-h-24 overflow-auto text-center text-[13px] text-foreground/90">{caption || copy.voice.listening}</p>
      <div className="flex items-center gap-2">
        <IconBtn label={muteMic ? copy.voice.unmuteMic : copy.voice.muteMic} onClick={() => setMuteMic((v) => !v)}>
          {muteMic ? <MicOff className="size-4" /> : <Mic className="size-4" />}
        </IconBtn>
        <IconBtn label={muteSpk ? copy.voice.unmuteSpk : copy.voice.muteSpk} onClick={() => setMuteSpk((v) => !v)}>
          {muteSpk ? <VolumeX className="size-4" /> : <Volume2 className="size-4" />}
        </IconBtn>
        <button type="button" className="grid size-10 place-items-center rounded-full bg-danger text-white" aria-label={copy.voice.hangup} onClick={() => { void endCall(); }}>
          <PhoneOff className="size-4" />
        </button>
      </div>
    </div>
  );
}

function IconBtn(props: { label: string; onClick: () => void; children: React.ReactNode }) {
  return (
    <button type="button" className={cn("grid size-9 place-items-center rounded-full bg-lift text-foreground")} aria-label={props.label} onClick={props.onClick}>
      {props.children}
    </button>
  );
}

function blobToB64(blob: Blob): Promise<string> {
  return new Promise((resolve, reject) => {
    const r = new FileReader();
    r.onload = () => {
      const s = String(r.result || "");
      const i = s.indexOf(",");
      resolve(i >= 0 ? s.slice(i + 1) : s);
    };
    r.onerror = () => reject(r.error);
    r.readAsDataURL(blob);
  });
}
