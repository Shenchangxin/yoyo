import { useEffect, useRef, useState } from "react";
import { Mic, MicOff, PhoneOff, Volume2, VolumeX } from "lucide-react";
import * as api from "../../lib/client";
import { useCopy } from "../../lib/i18n";
import { str } from "../../lib/normalize";
import { cn } from "../../lib/utils";

type Seg = { spoken?: string; subtitle?: string };

export function IdolCall(props: { paused?: boolean; approvals?: number; onHangup: () => void }) {
  const copy = useCopy();
  const c = copy.idol;
  const [phase, setPhase] = useState("listening");
  const [caption, setCaption] = useState("");
  const [setup, setSetup] = useState("");
  const [name, setName] = useState("");
  const [color, setColor] = useState("#c47b8a");
  const [portrait, setPortrait] = useState("");
  const [elapsed, setElapsed] = useState(0);
  const [muteMic, setMuteMic] = useState(false);
  const [muteSpk, setMuteSpk] = useState(false);
  const [textOnly, setTextOnly] = useState(false);
  const [phone, setPhone] = useState(false);
  const [work, setWork] = useState(false);
  const [draft, setDraft] = useState("");
  const idolId = useRef("");
  const alive = useRef(true);
  const stream = useRef<MediaStream | null>(null);
  const audio = useRef<HTMLAudioElement | null>(null);
  const stopSpeak = useRef(false);
  const connected = useRef(false);
  const started = useRef(Date.now());
  const ended = useRef(false);
  const hangupRef = useRef(props.onHangup);
  hangupRef.current = props.onHangup;
  const muteMicRef = useRef(false);
  muteMicRef.current = muteMic;
  const workRef = useRef(false);
  workRef.current = work;
  const typed = useRef("");

  useEffect(() => {
    const t = window.setInterval(() => setElapsed(Date.now() - started.current), 1000);
    return () => window.clearInterval(t);
  }, []);

  useEffect(() => {
    alive.current = true;
    void boot();
    return () => {
      alive.current = false;
      void finish(true);
    };
  }, []);

  useEffect(() => {
    if (props.paused) void finish(false);
  }, [props.paused]);

  async function boot() {
    tone(440, 180);
    void api.setCompanionCall(true);
    try {
      const snap = await api.idolCall("snapshot");
      const idol = snap?.pinned;
      if (!idol?.id) {
        setSetup(c.needPin);
        setPhase("off");
        return;
      }
      idolId.current = str(idol.id);
      setName(str(idol.name));
      setColor(str(idol.color) || "#c47b8a");
      setPortrait(str(snap.portrait));
      setTextOnly(!!snap.text_only);
      setPhone(!!snap.phone);
      setWork(snap.work !== false && idol.work_enabled !== false);
      stream.current = await navigator.mediaDevices.getUserMedia({ audio: true });
      tone(660, 120);
      connected.current = true;
      void loop();
    } catch (e: any) {
      setSetup(String(e?.message || e));
      setPhase("off");
    }
  }

  async function loop() {
    while (alive.current && !props.paused) {
      const queued = typed.current.trim();
      if (queued) {
        typed.current = "";
        await say(queued);
        continue;
      }
      if (muteMicRef.current) {
        await sleep(300);
        continue;
      }
      setPhase("listening");
      const heard = await listen(stream.current, () => !!typed.current.trim());
      if (!alive.current) return;
      if (!heard.trim()) continue;
      await say(heard);
    }
  }

  async function say(text: string) {
    setPhase("thinking");
    setCaption(text);
    stopSpeak.current = false;
    try {
      const res = await api.idolCall("turn", {
        idol_id: idolId.current,
        text,
        call: true,
        work: workRef.current,
      });
      if (!alive.current) return;
      if (res?.hung_up) setSetup(c.cooldown);
      const segs: Seg[] = res?.segments || [];
      setPhase("speaking");
      let played = 0;
      for (const seg of segs) {
        if (!alive.current || stopSpeak.current) {
          liveAudio?.pause();
          break;
        }
        setCaption(str(seg.subtitle || seg.spoken));
        if (!textOnly && !muteSpk) await speak(str(seg.spoken));
        else await sleep(700);
        if (stopSpeak.current) break;
        played += 1;
      }
      if (stopSpeak.current && res?.record_id) {
        void api.idolCall("truncate", { id: res.record_id, keep: played });
      }
      if (res?.work && res?.work_session) void pollWork(str(res.work_session));
    } catch (e: any) {
      const msg = String(e?.message || e);
      if (/cancel|context canceled/i.test(msg)) return;
      setSetup(msg === "cooldown" ? c.cooldown : msg === "ended" ? c.ended : msg);
      setPhase("listening");
    }
  }

  async function pollWork(session: string) {
    for (let i = 0; i < 40 && alive.current; i++) {
      await sleep(2000);
      try {
        const res = await api.idolCall("narrate", { idol_id: idolId.current, session });
        if (res?.pending) continue;
        if (res?.speak) {
          setCaption(str(res.caption || res.speak));
          if (!textOnly && !muteSpk) await speak(str(res.speak));
        }
        return;
      } catch {
        return;
      }
    }
  }

  async function finish(fromUnmount = false) {
    if (ended.current) return;
    ended.current = true;
    alive.current = false;
    stopSpeak.current = true;
    liveAudio?.pause();
    audio.current?.pause();
    stream.current?.getTracks().forEach((t) => t.stop());
    const sec = Math.round((Date.now() - started.current) / 1000);
    if (idolId.current) {
      void api.idolCall("call_end", { idol_id: idolId.current, seconds: sec, connected: connected.current });
    }
    tone(330, 160);
    void api.setCompanionCall(false);
    if (!fromUnmount) hangupRef.current();
  }

  async function speak(text: string) {
    if (!text.trim()) return;
    const out = await api.idolCall("speak", { idol_id: idolId.current, text });
    if (!out?.audio || stopSpeak.current) return;
    await play(`data:${out.content_type || "audio/mpeg"};base64,${out.audio}`, phone);
  }

  async function sendDraft() {
    const text = draft.trim();
    if (!text || !idolId.current) return;
    setDraft("");
    typed.current = text;
    stopSpeak.current = true;
    liveAudio?.pause();
    if (phase === "thinking") void api.idolCall("cancel");
  }

  const mm = String(Math.floor(elapsed / 60000)).padStart(2, "0");
  const ss = String(Math.floor((elapsed % 60000) / 1000)).padStart(2, "0");
  const phaseLabel = phase === "listening" ? c.listening : phase === "thinking" ? c.thinking : phase === "speaking" ? c.speaking : phase;

  return (
    <div className="flex h-full flex-col items-center justify-between px-4 py-3" data-testid="idol-call">
      <div className="text-[12px] text-muted">{name || c.title} · {mm}:{ss} · {phaseLabel}</div>
      <Face color={color} portrait={portrait} speaking={phase === "speaking"} />
      {props.approvals ? <div className="text-center text-[12px] text-warning">{c.approval}</div> : null}
      {setup ? <div className="text-center text-[12px] text-warning">{setup}</div> : null}
      <p className="max-h-24 overflow-auto text-center text-[13px] text-foreground/90">{caption || c.listening}</p>
      <label className="flex items-center gap-2 text-[11px] text-muted">
        <input type="checkbox" checked={work} onChange={(e) => setWork(e.target.checked)} />
        {c.work}
      </label>
      <form className="flex w-full gap-1" onSubmit={(e) => { e.preventDefault(); void sendDraft(); }}>
        <input
          className="h-8 min-w-0 flex-1 rounded-md bg-lift px-2 text-[12px]"
          value={draft}
          placeholder={c.type}
          onChange={(e) => setDraft(e.target.value)}
        />
      </form>
      <div className="flex items-center gap-2">
        <IconBtn label={muteMic ? c.unmuteMic : c.muteMic} onClick={() => setMuteMic((v) => !v)}>
          {muteMic ? <MicOff className="size-4" /> : <Mic className="size-4" />}
        </IconBtn>
        <IconBtn label={muteSpk ? c.unmuteSpk : c.muteSpk} onClick={() => setMuteSpk((v) => !v)}>
          {muteSpk ? <VolumeX className="size-4" /> : <Volume2 className="size-4" />}
        </IconBtn>
        <button type="button" className="grid size-10 place-items-center rounded-full bg-danger text-white" aria-label={c.hangup} onClick={() => { void finish(false); }}>
          <PhoneOff className="size-4" />
        </button>
      </div>
    </div>
  );
}

function Face(props: { color: string; portrait: string; speaking: boolean }) {
  return (
    <div
      className={cn("grid size-36 place-items-center overflow-hidden rounded-full", props.speaking && "scale-105")}
      style={{ background: props.color }}
    >
      {props.portrait ? (
        <img alt="" className="size-full object-cover" src={`data:image/png;base64,${props.portrait}`} />
      ) : null}
    </div>
  );
}

function IconBtn(props: { label: string; onClick: () => void; children: React.ReactNode }) {
  return (
    <button type="button" className="grid size-9 place-items-center rounded-full bg-lift text-foreground" aria-label={props.label} onClick={props.onClick}>
      {props.children}
    </button>
  );
}

function sleep(ms: number) {
  return new Promise((r) => setTimeout(r, ms));
}

async function listen(stream: MediaStream | null, stop?: () => boolean): Promise<string> {
  if (!stream) return "";
  const chunks: Blob[] = [];
  const mr = new MediaRecorder(stream);
  mr.ondataavailable = (e) => { if (e.data.size) chunks.push(e.data); };
  const done = new Promise<void>((resolve) => { mr.onstop = () => resolve(); });
  const ctx = new AudioContext();
  const src = ctx.createMediaStreamSource(stream);
  const analyser = ctx.createAnalyser();
  analyser.fftSize = 512;
  src.connect(analyser);
  const data = new Uint8Array(analyser.fftSize);
  mr.start();
  const t0 = Date.now();
  let heard = false;
  let silence = 0;
  while (Date.now() - t0 < 20000) {
    analyser.getByteTimeDomainData(data);
    let sum = 0;
    for (const v of data) {
      const x = (v - 128) / 128;
      sum += x * x;
    }
    const rms = Math.sqrt(sum / data.length);
    if (rms > 0.025) {
      heard = true;
      silence = 0;
    } else if (heard) silence += 80;
    if (heard && silence > 700) break;
    if (!heard && Date.now() - t0 > 8000) break;
    if (stop?.()) break;
    await sleep(80);
  }
  if (mr.state === "recording") mr.stop();
  await done;
  await ctx.close().catch(() => {});
  if (stop?.() || !heard) return "";
  const blob = new Blob(chunks, { type: mr.mimeType || "audio/webm" });
  const b64 = await blobToB64(blob);
  const out = await api.idolCall("transcribe", { filename: "audio.webm", audio: b64 });
  return str(out?.text);
}

function play(url: string, phone: boolean): Promise<void> {
  return new Promise((resolve) => {
    const el = new Audio(url);
    liveAudio = el;
    const done = () => { if (liveAudio === el) liveAudio = null; resolve(); };
    if (!phone) {
      el.onended = done;
      el.onerror = done;
      void el.play().catch(done);
      return;
    }
    const ctx = new AudioContext();
    const node = ctx.createMediaElementSource(el);
    const filter = ctx.createBiquadFilter();
    filter.type = "lowpass";
    filter.frequency.value = 1800;
    node.connect(filter);
    filter.connect(ctx.destination);
    el.onended = () => { void ctx.close(); done(); };
    el.onerror = () => { void ctx.close(); done(); };
    void el.play().catch(done);
  });
}

let liveAudio: HTMLAudioElement | null = null;

function tone(freq: number, ms: number) {
  const ctx = new AudioContext();
  const o = ctx.createOscillator();
  const g = ctx.createGain();
  o.frequency.value = freq;
  o.connect(g);
  g.connect(ctx.destination);
  g.gain.value = 0.04;
  o.start();
  o.stop(ctx.currentTime + ms / 1000);
  o.onended = () => { void ctx.close(); };
}

function blobToB64(blob: Blob): Promise<string> {
  return new Promise((resolve, reject) => {
    const r = new FileReader();
    r.onload = () => {
      const s = String(r.result || "");
      resolve(s.includes(",") ? s.slice(s.indexOf(",") + 1) : s);
    };
    r.onerror = () => reject(r.error);
    r.readAsDataURL(blob);
  });
}
