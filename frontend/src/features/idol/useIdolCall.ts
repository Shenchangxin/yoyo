import { useEffect, useRef, useState } from "react";
import * as api from "../../lib/client";
import { useCopy } from "../../lib/i18n";
import { str } from "../../lib/normalize";
import { idolError, type IdolErrors } from "./model";
import { standMood } from "./stand";

export type CallPhase = "dialing" | "listening" | "thinking" | "speaking" | "ended" | "off";

export type CallWho = { name: string; color: string; portrait: string; kind: string; stand: string };
type Run = {
  stop: boolean;
  closed: boolean;
  hold: boolean;
  idolId: string;
  connected: boolean;
  started: number;
  stream: MediaStream | null;
  typed: string;
  companion: boolean;
};

type Opts = {
  active: boolean;
  idolId?: string;
  companion?: boolean;
  name?: string;
  color?: string;
  portrait?: string;
  kind?: string;
  stand?: string;
  work?: boolean;
  textOnly?: boolean;
  phone?: boolean;
  onEnded?: () => void;
};

let primed: AudioContext | null = null;
let liveAudio: HTMLAudioElement | null = null;

export function primeCallAudio() {
  const Ctx = window.AudioContext || (window as unknown as { webkitAudioContext?: typeof AudioContext }).webkitAudioContext;
  if (!Ctx) return;
  if (!primed) primed = new Ctx();
  void primed.resume();
}

export function useIdolCall(opts: Opts) {
  const copy = useCopy();
  const c = copy.idol;
  const [phase, setPhase] = useState<CallPhase>(opts.active ? "dialing" : "off");
  const [caption, setCaption] = useState("");
  const [notice, setNotice] = useState("");
  const [elapsed, setElapsed] = useState(0);
  const [muteMic, setMuteMic] = useState(false);
  const [muteSpk, setMuteSpk] = useState(false);
  const [textOnly, setTextOnly] = useState(!!opts.textOnly);
  const [draft, setDraft] = useState("");
  const [emotion, setEmotion] = useState("calm");
  const [who, setWho] = useState<CallWho>({
    name: opts.name || "",
    color: opts.color || "#c47b8a",
    portrait: opts.portrait || "",
    kind: opts.kind || "",
    stand: opts.stand || "",
  });
  const runRef = useRef<Run | null>(null);
  const life = useRef(0);
  const onEnded = useRef(opts.onEnded);
  onEnded.current = opts.onEnded;
  const workRef = useRef(!!opts.work);
  workRef.current = !!opts.work;
  const phoneRef = useRef(!!opts.phone);
  phoneRef.current = !!opts.phone;
  const muteMicRef = useRef(false);
  muteMicRef.current = muteMic;
  const muteSpkRef = useRef(false);
  muteSpkRef.current = muteSpk;
  const textOnlyRef = useRef(textOnly);
  textOnlyRef.current = textOnly;
  const phaseRef = useRef(phase);
  phaseRef.current = phase;
  const stopSpeak = useRef(false);
  const errors = useRef<IdolErrors>(c);
  errors.current = c;
  const callKey = `${opts.active ? 1 : 0}:${opts.idolId || ""}:${opts.companion ? 1 : 0}`;
  const [track, setTrack] = useState(callKey);
  if (track !== callKey) {
    setTrack(callKey);
    setPhase(opts.active ? "dialing" : "off");
    setNotice("");
    setCaption("");
    setElapsed(0);
    setDraft("");
    setEmotion("calm");
  }

  useEffect(() => {
    if (opts.companion) return;
    setWho({
      name: opts.name || "",
      color: opts.color || "#c47b8a",
      portrait: opts.portrait || "",
      kind: opts.kind || "",
      stand: opts.stand || "",
    });
  }, [opts.companion, opts.name, opts.color, opts.portrait, opts.kind, opts.stand]);

  useEffect(() => {
    if (!opts.active) return;
    const mine = ++life.current;
    const run: Run = {
      stop: false,
      closed: false,
      hold: false,
      idolId: opts.companion ? "" : (opts.idolId || ""),
      connected: false,
      started: Date.now(),
      stream: null,
      typed: "",
      companion: !!opts.companion,
    };
    runRef.current = run;
    setPhase("dialing");
    setNotice("");
    setCaption("");
    setElapsed(0);
    setTextOnly(!!opts.textOnly);
    void boot(run);
    const tick = window.setInterval(() => {
      if (!run.stop) setElapsed(Date.now() - run.started);
    }, 500);
    return () => {
      window.clearInterval(tick);
      run.stop = true;
      run.stream?.getTracks().forEach((t) => t.stop());
      run.stream = null;
      queueMicrotask(() => {
        if (life.current !== mine) return;
        stopRun(run, false);
      });
    };
    // Restart only when the call itself starts or the person changes.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [opts.active, opts.idolId, opts.companion]);

  async function boot(run: Run) {
    if (run.stop || run.closed) return;
    tone(440, 180);
    if (run.companion) void api.setCompanionCall(true);
    try {
      if (run.companion) {
        const snap = await api.idolCall("snapshot");
        if (run.stop) return;
        const idol = snap?.pinned;
        if (!idol?.id) {
          setNotice(c.needPin);
          setPhase("off");
          return;
        }
        run.idolId = str(idol.id);
        setWho({
          name: str(idol.name),
          color: str(idol.color) || "#c47b8a",
          portrait: str(snap.portrait),
          kind: str(idol.kind || idol.Kind),
          stand: str(idol.outfit_id || idol.OutfitID),
        });
        setTextOnly(!!snap.text_only);
        textOnlyRef.current = !!snap.text_only;
        phoneRef.current = !!snap.phone;
      } else if (!run.idolId) {
        setNotice(c.missing);
        setPhase("off");
        return;
      }
      if (run.stop) return;
      try {
        run.stream = await navigator.mediaDevices.getUserMedia({ audio: true });
      } catch {
        if (run.stop) return;
        setTextOnly(true);
        textOnlyRef.current = true;
        setNotice(c.micDenied);
      }
      if (run.stop) {
        run.stream?.getTracks().forEach((t) => t.stop());
        return;
      }
      tone(660, 120);
      run.connected = true;
      setPhase("listening");
      void loop(run);
    } catch (e) {
      if (run.stop) return;
      setNotice(idolError(e, errors.current));
      setPhase("off");
    }
  }

  async function loop(run: Run) {
    while (!run.stop) {
      if (run.hold) {
        await sleep(240);
        continue;
      }
      const queued = run.typed.trim();
      if (queued) {
        run.typed = "";
        await say(run, queued);
        continue;
      }
      if (textOnlyRef.current || muteMicRef.current || !run.stream) {
        setPhase((p) => (p === "ended" || p === "off" ? p : "listening"));
        await sleep(240);
        continue;
      }
      setPhase("listening");
      const heard = await listen(run.stream, () => run.stop || !!run.typed.trim());
      if (run.stop) return;
      if (!heard.trim()) continue;
      await say(run, heard);
    }
  }

  async function say(run: Run, text: string) {
    if (run.stop) return;
    setPhase("thinking");
    setCaption(text);
    stopSpeak.current = false;
    try {
      const res = await api.idolCall("turn", {
        idol_id: run.idolId,
        text,
        call: true,
        work: workRef.current,
      });
      if (run.stop) return;
      setEmotion(standMood(str(res?.emotion)));
      const segs: { spoken?: string; subtitle?: string }[] = res?.segments || [];
      if (res?.hung_up || res?.ended) setNotice(res?.hung_up ? errors.current.cooldown : errors.current.ended);
      setPhase("speaking");
      let played = 0;
      for (const seg of segs) {
        if (run.stop || stopSpeak.current) {
          liveAudio?.pause();
          break;
        }
        setCaption(str(seg.subtitle || seg.spoken));
        if (!textOnlyRef.current && !muteSpkRef.current) {
          try { await speak(run.idolId, str(seg.spoken), phoneRef.current); } catch { await sleep(400); }
        } else await sleep(700);
        if (stopSpeak.current || run.stop) break;
        played += 1;
      }
      if (stopSpeak.current && res?.record_id) {
        void api.idolCall("truncate", { id: res.record_id, keep: played });
      }
      if (res?.work && res?.work_session) void pollWork(run, str(res.work_session));
      if (res?.hung_up || res?.ended) {
        setPhase("ended");
        run.hold = true;
        run.stream?.getTracks().forEach((t) => t.stop());
        run.stream = null;
        return;
      }
      if (!run.stop) setPhase("listening");
    } catch (e: any) {
      const msg = String(e?.message || e);
      if (run.stop || /cancel|context canceled/i.test(msg)) return;
      setNotice(idolError(e, errors.current));
      setPhase("listening");
    }
  }

  async function pollWork(run: Run, session: string) {
    for (let i = 0; i < 40 && !run.stop; i++) {
      await sleep(2000);
      if (run.stop) return;
      try {
        const res = await api.idolCall("narrate", { idol_id: run.idolId, session });
        if (res?.pending) continue;
        if (res?.speak) {
          setCaption(str(res.caption || res.speak));
          if (!textOnlyRef.current && !muteSpkRef.current) {
            try { await speak(run.idolId, str(res.speak), phoneRef.current); } catch { /* caption is enough */ }
          }
        }
        return;
      } catch {
        return;
      }
    }
  }

  async function stopRun(run: Run, notify: boolean) {
    if (run.closed) return;
    run.closed = true;
    run.stop = true;
    stopSpeak.current = true;
    liveAudio?.pause();
    run.stream?.getTracks().forEach((t) => t.stop());
    run.stream = null;
    const sec = Math.round((Date.now() - run.started) / 1000);
    if (run.idolId) {
      void api.idolCall("call_end", { idol_id: run.idolId, seconds: sec, connected: run.connected });
    }
    void api.idolCall("cancel").catch(() => {});
    tone(330, 160);
    if (run.companion) {
      try { await api.setCompanionCall(false); } catch { /* the pet window still returns */ }
    }
    if (notify) onEnded.current?.();
  }

  function hangup() {
    const run = runRef.current;
    if (!run) {
      onEnded.current?.();
      return;
    }
    stopRun(run, true);
  }

  function sendDraft() {
    const text = draft.trim();
    if (!text) return;
    setDraft("");
    const run = runRef.current;
    if (!run || run.stop) return;
    run.typed = text;
    stopSpeak.current = true;
    liveAudio?.pause();
    if (phaseRef.current === "thinking") void api.idolCall("cancel");
  }

  return {
    phase, caption, notice, elapsed, muteMic, setMuteMic, muteSpk, setMuteSpk,
    textOnly, draft, setDraft, sendDraft, hangup, who, emotion,
  };
}

async function speak(idolId: string, text: string, phone: boolean) {
  if (!text.trim()) return;
  const out = await api.idolCall("speak", { idol_id: idolId, text });
  if (!out?.audio) return;
  await play(`data:${out.content_type || "audio/mpeg"};base64,${out.audio}`, phone);
}

async function listen(stream: MediaStream | null, stop?: () => boolean): Promise<string> {
  if (!stream) return "";
  const chunks: Blob[] = [];
  let mr: MediaRecorder;
  try {
    mr = new MediaRecorder(stream);
  } catch {
    return "";
  }
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
    if (stop?.()) break;
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
    await sleep(80);
  }
  if (mr.state === "recording") mr.stop();
  await done;
  await ctx.close().catch(() => {});
  if (stop?.() || !heard) return "";
  const blob = new Blob(chunks, { type: mr.mimeType || "audio/webm" });
  const b64 = await blobToB64(blob);
  try {
    const out = await api.idolCall("transcribe", { filename: "audio.webm", audio: b64 });
    return str(out?.text);
  } catch {
    return "";
  }
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
    const ctx = primed || new AudioContext();
    const node = ctx.createMediaElementSource(el);
    const filter = ctx.createBiquadFilter();
    filter.type = "lowpass";
    filter.frequency.value = 1800;
    node.connect(filter);
    filter.connect(ctx.destination);
    el.onended = () => { if (ctx !== primed) void ctx.close(); done(); };
    el.onerror = () => { if (ctx !== primed) void ctx.close(); done(); };
    void el.play().catch(done);
  });
}

function tone(freq: number, ms: number) {
  const ctx = primed || new AudioContext();
  const o = ctx.createOscillator();
  const g = ctx.createGain();
  o.frequency.value = freq;
  o.connect(g);
  g.connect(ctx.destination);
  g.gain.setValueAtTime(0.04, ctx.currentTime);
  o.start();
  o.stop(ctx.currentTime + ms / 1000);
  o.onended = () => { if (ctx !== primed) void ctx.close(); };
}

function sleep(ms: number) {
  return new Promise((r) => setTimeout(r, ms));
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

export function phoneFrom(flag: boolean) {
  return flag;
}
