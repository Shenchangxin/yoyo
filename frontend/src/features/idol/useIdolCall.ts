/**
 * 拥有一通偶像通话的阶段、麦克风、听写、播放和轮流说话的规则。不拥有立绘绘制。回合、挂断、截断和陪伴窗开关会改服务端状态。
 * 一轮话在服务端后台跑，这里用 turn_next 长轮询拿写完的段，每段一到就去合成，按顺序念出来，不等整段回复。
 * 她开口之前用户又说了或打了字，是补充：这一轮丢掉，连同补充的话重新回一次。她开口之后，是打断：她停下，只记下用户听到的那几段，
 * 用户的话成为下一轮。开口插话时，她正在出声就要求更响、更长的声音，免得外放的回声把她自己打断。
 */
import { useEffect, useRef, useState } from "react";
import * as api from "../../lib/client";
import { useCopy } from "../../lib/i18n";
import { asArray, str } from "../../lib/normalize";
import { idolError, isCooling, type IdolErrors } from "./model";
import { cueMood, standMood } from "./stand";

/** CallPhase 是通话阶段：拨号、听、想、说、结束或关掉。 */
export type CallPhase = "dialing" | "listening" | "thinking" | "speaking" | "ended" | "off";

/** CallWho 是这一通对面的名字、颜色、肖像、种类和服装。 */
export type CallWho = { name: string; color: string; portrait: string; kind: string; stand: string };

/** Seg 是服务端吐出的一段：口语、字幕和语气提示。 */
type Seg = { spoken?: string; subtitle?: string; cue?: string };

/** Spoken 是排队等她念的一段话，比如办完事之后的转述。 */
type Spoken = { segs: Seg[] };

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
  callId: string;
  turnId: string;
  bargeIn: boolean;
  ttsWarned: boolean;
  asrFails: number;
  queue: Spoken[];
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
  bargeIn?: boolean;
  onEnded?: () => void;
};

/** Outcome 是一次回复的去向：自然说完、被补充（重新回一次）或被打断（用户的话成为下一轮）。 */
type Outcome = { kind: "done" } | { kind: "followOn"; text: string } | { kind: "interrupt"; text: string };

/** 安静时一句话算开口的音量（均方根）。 */
const SPEECH_RMS = 0.025;
/** 她正在出声时，开口插话要达到的音量。 */
const BARGE_RMS = 0.07;
/** 她正在出声时，插话的声音要持续多久才算数。 */
const BARGE_MS = 280;

let primed: AudioContext | null = null;
let liveAudio: HTMLAudioElement | null = null;
let playing = false;

/** primeCallAudio 提前建好并恢复音频上下文，供之后播放使用。 */
export function primeCallAudio() {
  const Ctx = window.AudioContext || (window as unknown as { webkitAudioContext?: typeof AudioContext }).webkitAudioContext;
  if (!Ctx) return;
  if (!primed) primed = new Ctx();
  void primed.resume();
}

/** useIdolCall 跑一通通话：听、送回合、边收边念，并在结束时写回服务端。opts 读取 active、textOnly、name、color、portrait、kind 等。 */
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
  const stopSpeak = useRef(false);
  const errors = useRef<IdolErrors>(c);
  errors.current = c;
  const words = useRef(c);
  words.current = c;
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
      callId: newCallId(),
      turnId: "",
      bargeIn: opts.bargeIn !== false,
      ttsWarned: false,
      asrFails: 0,
      queue: [],
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
    // 只在通话开始或对方换人时重跑。
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [opts.active, opts.idolId, opts.companion]);

  /** 接通这一通：陪伴窗先取置顶偶像，再要麦克风，然后进入聆听。 */
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
        run.bargeIn = snap?.barge_in !== false;
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
        if (isCooling(str(snap?.affinity?.cooldown_until))) {
          setNotice(c.cooldown);
          setPhase("off");
          return;
        }
      } else if (!run.idolId) {
        setNotice(c.missing);
        setPhase("off");
        return;
      }
      if (run.stop) return;
      try {
        run.stream = await navigator.mediaDevices.getUserMedia({
          audio: { echoCancellation: true, noiseSuppression: true, autoGainControl: true },
        });
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
      void api.idolCall("call_start", { idol_id: run.idolId }).catch(() => {});
      setPhase("listening");
      void loop(run);
    } catch (e) {
      if (run.stop) return;
      setNotice(idolError(e, errors.current));
      setPhase("off");
    }
  }

  /** 在停掉之前反复：先念排队的话，再发已键入的文字，否则听一句。 */
  async function loop(run: Run) {
    while (!run.stop) {
      if (run.hold) {
        await sleep(240);
        continue;
      }
      const next = run.queue.shift();
      if (next) {
        await speakAll(run, next.segs);
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
      const heard = await capture(run.stream, {
        idolId: run.idolId,
        stop: () => run.stop || !!run.typed.trim() || run.queue.length > 0,
        threshold: () => SPEECH_RMS,
        onsetMs: 80,
        maxWaitMs: 8000,
      });
      if (run.stop) return;
      if (heard.error) {
        hearFailed(run, heard.error);
        continue;
      }
      run.asrFails = 0;
      if (!heard.text.trim()) continue;
      await say(run, heard.text);
    }
  }

  /** 听写失败时如实说出来；识别没配好或连着失败时，这通电话改用打字。 */
  function hearFailed(run: Run, reason: string) {
    run.asrFails += 1;
    const broken = /not configured|401|403|404|unauthor|forbidden|invalid api key|no key/i.test(reason);
    if (broken || run.asrFails >= 3) {
      setTextOnly(true);
      textOnlyRef.current = true;
      setNotice(words.current.asrOff.replace("{reason}", shortReason(reason)));
      return;
    }
    setNotice(words.current.asrFailed.replace("{reason}", shortReason(reason)));
  }

  /** 回这一句，补充时连同补充的话重新回，打断时把用户的话排成下一轮。 */
  async function say(run: Run, first: string) {
    let text = first;
    for (let round = 0; round < 6 && !run.stop; round++) {
      const out = await reply(run, text);
      if (out.kind === "followOn") {
        if (out.text.trim()) text = `${text}\n${out.text.trim()}`;
        continue;
      }
      if (out.kind === "interrupt" && out.text.trim()) run.typed = out.text.trim();
      return;
    }
  }

  /** 送一轮回复：边收边念；中途用户开口或打字时按她是否已经开口，决定补充还是打断。 */
  async function reply(run: Run, text: string): Promise<Outcome> {
    if (run.stop) return { kind: "done" };
    setPhase("thinking");
    setCaption(text);
    stopSpeak.current = false;
    let id = "";
    try {
      const open = await api.idolCall("turn_open", {
        idol_id: run.idolId, text, call: true, call_id: run.callId, work: workRef.current,
      });
      id = str(open?.id);
    } catch (e) {
      if (!run.stop) {
        setNotice(idolError(e, errors.current));
        setPhase("listening");
      }
      return { kind: "done" };
    }
    if (!id || run.stop) return { kind: "done" };
    run.turnId = id;
    const voiced = !textOnlyRef.current && !muteSpkRef.current;
    const segs: Seg[] = [];
    const audio: Promise<string | null>[] = [];
    let done = false;
    let result: any = null;
    let error = "";
    let spoke = false;
    let played = 0;
    /** 回调里会改写的几样放在一个对象里：用户的话是补充还是打断、他说了什么、这一轮是否已经收尾。 */
    const turn: { decided: "" | "followOn" | "interrupt"; input: Promise<string> | null; closing: boolean } = {
      decided: "", input: null, closing: false,
    };
    let monitorOff = false;

    /** 用户开口或打了字：她还没开口、回复也还没记下时是补充，丢掉这一轮；开口了或已经记下了就是打断，她停下，只记写完的段。
     * 这一轮已经说完时不接，留给下一轮。 */
    const take = (p: Promise<string>) => {
      if (turn.input || turn.closing) return;
      turn.input = p;
      const late = spoke || done;
      turn.decided = late ? "interrupt" : "followOn";
      void api.idolCall(late ? "turn_stop" : "turn_cancel", { id }).catch(() => {});
      stopSpeak.current = true;
      liveAudio?.pause();
    };

    const fetching = (async () => {
      while (!run.stop && !done) {
        try {
          const v = await api.idolCall("turn_next", { id, after: segs.length, wait_ms: 15000 });
          if (v?.emotion && segs.length === 0) setEmotion(standMood(str(v.emotion)));
          for (const raw of asArray(v?.segments)) {
            const seg: Seg = { spoken: str(raw?.spoken), subtitle: str(raw?.subtitle), cue: str(raw?.cue) };
            segs.push(seg);
            audio.push(voiced ? fetchSpeech(run, seg) : Promise.resolve(null));
          }
          if (v?.done) {
            done = true;
            result = v.result || null;
            error = str(v.error);
          }
        } catch (e) {
          done = true;
          error = String((e as Error)?.message || e);
        }
      }
    })();

    const listening = run.bargeIn && voiced && !!run.stream && !muteMicRef.current;
    let monitorHeard = false;
    let monitor: Promise<{ text: string; error?: string }> = Promise.resolve({ text: "" });
    if (listening && run.stream) {
      let resolveInput: (t: string) => void = () => {};
      const said = new Promise<string>((r) => { resolveInput = r; });
      monitor = capture(run.stream, {
        idolId: run.idolId,
        stop: () => monitorOff || run.stop,
        threshold: () => (playing ? BARGE_RMS : SPEECH_RMS * 1.4),
        onsetMs: playing ? BARGE_MS : 180,
        maxWaitMs: 0,
        onStart: () => {
          monitorHeard = true;
          take(said);
        },
      }).then((r) => {
        if (r.error) hearFailed(run, r.error);
        resolveInput(r.text || "");
        return r;
      });
    }

    void (async () => {
      while (!run.stop && !turn.decided && !turn.closing) {
        const t = run.typed.trim();
        if (t) {
          run.typed = "";
          take(Promise.resolve(t));
          return;
        }
        await sleep(120);
      }
    })();

    let i = 0;
    while (!run.stop && !turn.decided) {
      if (i < segs.length) {
        const seg = segs[i];
        const url = await audio[i];
        if (run.stop || turn.decided) break;
        spoke = true;
        setPhase("speaking");
        setCaption(str(seg.subtitle || seg.spoken));
        const mood = cueMood(seg.cue);
        if (mood) setEmotion(mood);
        if (url) await play(url, phoneRef.current);
        else await sleep(readPace(seg));
        if (run.stop || turn.decided || stopSpeak.current) break;
        played = i + 1;
        i += 1;
        continue;
      }
      if (done) break;
      await sleep(60);
    }
    if (!turn.decided) turn.closing = true;
    if (turn.decided === "followOn") {
      monitorOff = true;
      const more = await (turn.input ?? Promise.resolve(""));
      await fetching;
      if (result) {
        afterReply(run, result);
        return { kind: "interrupt", text: more };
      }
      return { kind: "followOn", text: more };
    }
    if (turn.decided === "interrupt") {
      await fetching;
      monitorOff = true;
      if (result?.record_id && played > 0 && played < asArray(result?.segments).length) {
        void api.idolCall("truncate", { id: result.record_id, keep: played }).catch(() => {});
      }
      afterReply(run, result);
      const more = await (turn.input ?? Promise.resolve(""));
      return { kind: "interrupt", text: more };
    }
    await fetching;
    monitorOff = true;
    if (monitorHeard && !turn.input) {
      const late = await monitor;
      if (late.text.trim() && !run.stop) run.typed = late.text.trim();
    }
    if (run.stop) return { kind: "done" };
    if (error && error !== "canceled" && error !== "stopped") {
      setNotice(idolError(new Error(error), errors.current));
      setPhase("listening");
      return { kind: "done" };
    }
    afterReply(run, result);
    return { kind: "done" };
  }

  /** 一轮说完之后：她要挂断就结束这一通；拍了照片或交出了事情，就记着之后告诉用户。 */
  function afterReply(run: Run, result: any) {
    if (!result || run.stop) return;
    if (result.photo_pending || result.photo_id) setNotice(words.current.photoInCall);
    if (result.work && result.work_session) void pollWork(run, str(result.work_session), str(result.record_id));
    if (result.hung_up || result.ended) {
      setNotice(result.hung_up ? words.current.hungUp : errors.current.ended);
      setPhase("ended");
      run.hold = true;
      run.stream?.getTracks().forEach((t) => t.stop());
      run.stream = null;
      return;
    }
    if (!run.stop) setPhase("listening");
  }

  /** 按顺序念几段排队的话（例如办完事的转述），不接受补充。 */
  async function speakAll(run: Run, list: Seg[]) {
    const voiced = !textOnlyRef.current && !muteSpkRef.current;
    const audio = list.map((s) => (voiced ? fetchSpeech(run, s) : Promise.resolve(null)));
    stopSpeak.current = false;
    for (let i = 0; i < list.length && !run.stop; i++) {
      const url = await audio[i];
      if (run.stop || stopSpeak.current) break;
      setPhase("speaking");
      setCaption(str(list[i].subtitle || list[i].spoken));
      const mood = cueMood(list[i].cue);
      if (mood) setEmotion(mood);
      if (url) await play(url, phoneRef.current);
      else await sleep(readPace(list[i]));
    }
    if (!run.stop && !run.hold) setPhase("listening");
  }

  /** 合成一段。失败时第一次如实提示，之后只看字幕。 */
  async function fetchSpeech(run: Run, seg: Seg): Promise<string | null> {
    const text = str(seg.spoken).trim();
    if (!text) return null;
    try {
      const out = await api.idolCall("speak", { idol_id: run.idolId, text, cue: str(seg.cue) });
      if (!out?.audio) return null;
      return `data:${out.content_type || "audio/mpeg"};base64,${out.audio}`;
    } catch (e) {
      if (!run.ttsWarned && !run.stop) {
        run.ttsWarned = true;
        setNotice(words.current.ttsFailed.replace("{reason}", shortReason(String((e as Error)?.message || e))));
      }
      return null;
    }
  }

  /** 等她交给助手的事办完，把她的转述排进来念。挂断后服务端照样会把转述写进对话。 */
  async function pollWork(run: Run, session: string, recordId: string) {
    while (!run.stop) {
      try {
        const res = await api.idolCall("narrate", { idol_id: run.idolId, session, record_id: recordId, wait_ms: 20000 });
        if (res?.pending) continue;
        const segs = asArray(res?.segments).map((s: any) => ({ spoken: str(s?.spoken), subtitle: str(s?.subtitle), cue: str(s?.cue) }));
        if (segs.length && !run.stop) run.queue.push({ segs });
        return;
      } catch {
        return;
      }
    }
  }

  /** 停掉麦克风、播放和这一轮，并写下通话结束。notify 为真时才通知外面。 */
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
    if (run.turnId) void api.idolCall("turn_cancel", { id: run.turnId }).catch(() => {});
    tone(330, 160);
    if (run.companion) {
      try { await api.setCompanionCall(false); } catch { /* 陪伴窗仍会回到原处 */ }
    }
    if (notify) onEnded.current?.();
  }

  /** 挂断当前这一通，没有进行中的就直接通知结束。 */
  function hangup() {
    const run = runRef.current;
    if (!run) {
      onEnded.current?.();
      return;
    }
    stopRun(run, true);
  }

  /** 把输入框里的文字交给这一通：她在想时是补充，在说时打断她，空闲时成为下一轮。 */
  function sendDraft() {
    const text = draft.trim();
    if (!text) return;
    setDraft("");
    const run = runRef.current;
    if (!run || run.stop) return;
    run.typed = text;
  }

  return {
    phase, caption, notice, elapsed, muteMic, setMuteMic, muteSpk, setMuteSpk,
    textOnly, draft, setDraft, sendDraft, hangup, who, emotion,
  };
}

/** CaptureOpts：stop 在开口之前叫停；threshold 是此刻算开口的音量；onsetMs 是声音要持续多久；maxWaitMs 为 0 时一直等到 stop。idolId 用来把她的名字当热词交给识别。 */
type CaptureOpts = {
  idolId?: string;
  stop: () => boolean;
  threshold: () => number;
  onsetMs: number;
  maxWaitMs: number;
  onStart?: () => void;
};

/** 录一句话再交给转写：先等到开口（声音够响、够长），叫一下 onStart，再录到停顿 700 毫秒或满 20 秒。
 * 录音从一开始就在录，所以开口那一下不会被切掉。开口前叫停或等不到开口时返回空文字；转写失败时带上 error。 */
async function capture(stream: MediaStream, opts: CaptureOpts): Promise<{ text: string; error?: string }> {
  const chunks: Blob[] = [];
  let mr: MediaRecorder;
  try {
    mr = new MediaRecorder(stream);
  } catch {
    return { text: "" };
  }
  mr.ondataavailable = (e) => { if (e.data.size) chunks.push(e.data); };
  const done = new Promise<void>((resolve) => { mr.onstop = () => resolve(); });
  const ctx = new AudioContext();
  const src = ctx.createMediaStreamSource(stream);
  const analyser = ctx.createAnalyser();
  analyser.fftSize = 512;
  src.connect(analyser);
  const data = new Uint8Array(analyser.fftSize);
  mr.start(250);
  const t0 = Date.now();
  let loud = 0;
  let heard = false;
  let heardAt = 0;
  let silence = 0;
  for (;;) {
    if (!heard && opts.stop()) break;
    analyser.getByteTimeDomainData(data);
    let sum = 0;
    for (const v of data) {
      const x = (v - 128) / 128;
      sum += x * x;
    }
    const rms = Math.sqrt(sum / data.length);
    if (!heard) {
      loud = rms > opts.threshold() ? loud + 60 : 0;
      if (loud >= opts.onsetMs) {
        heard = true;
        heardAt = Date.now();
        opts.onStart?.();
      } else if (opts.maxWaitMs > 0 && Date.now() - t0 > opts.maxWaitMs) break;
    } else {
      silence = rms > SPEECH_RMS ? 0 : silence + 60;
      if (silence > 700 || Date.now() - heardAt > 20000) break;
    }
    await sleep(60);
  }
  if (mr.state === "recording") mr.stop();
  await done;
  await ctx.close().catch(() => {});
  if (!heard) return { text: "" };
  const blob = new Blob(chunks, { type: mr.mimeType || "audio/webm" });
  const b64 = await blobToB64(blob);
  try {
    const out = await api.idolCall("transcribe", { filename: fileFor(mr.mimeType), audio: b64, idol_id: opts.idolId || "" });
    return { text: str(out?.text) };
  } catch (e) {
    return { text: "", error: String((e as Error)?.message || e) };
  }
}

/** 按录音格式给上传文件名，转写服务靠扩展名认格式。 */
function fileFor(mime: string): string {
  if (/ogg/i.test(mime)) return "audio.ogg";
  if (/mp4|aac/i.test(mime)) return "audio.m4a";
  return "audio.webm";
}

/** 播放一段音频；电话音质时只留 300 到 3400 赫兹，像隔着听筒。 */
function play(url: string, phone: boolean): Promise<void> {
  return new Promise((resolve) => {
    const el = new Audio(url);
    liveAudio = el;
    playing = true;
    let ctx: AudioContext | null = null;
    let over = false;
    /** 这段播完、被停下或失败就结束等待，并清掉当前播放。暂停和播完会各来一次，只算第一次。 */
    const done = () => {
      if (over) return;
      over = true;
      if (liveAudio === el) {
        liveAudio = null;
        playing = false;
      }
      if (ctx && ctx !== primed) void ctx.close();
      resolve();
    };
    el.onended = done;
    el.onerror = done;
    el.onpause = done;
    if (phone) {
      ctx = primed || new AudioContext();
      const node = ctx.createMediaElementSource(el);
      const low = ctx.createBiquadFilter();
      low.type = "highpass";
      low.frequency.value = 300;
      const high = ctx.createBiquadFilter();
      high.type = "lowpass";
      high.frequency.value = 3400;
      node.connect(low);
      low.connect(high);
      high.connect(ctx.destination);
    }
    void el.play().catch(done);
  });
}

/** 只看文字时一段字幕停留多久：按每秒 5 个字，最短 1.5 秒，最长 8 秒。 */
function readPace(seg: Seg): number {
  const n = Array.from(str(seg.subtitle || seg.spoken)).length;
  return Math.min(8000, Math.max(1500, Math.round(n / 5 * 1000)));
}

/** 用振荡器响一声提示音。 */
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

/** 等待一段时间。 */
function sleep(ms: number) {
  return new Promise((r) => setTimeout(r, ms));
}

/** 给这一通一个 id，服务端按它累计这通电话里的告别。 */
function newCallId(): string {
  const c = globalThis.crypto as Crypto | undefined;
  if (c && typeof c.randomUUID === "function") return c.randomUUID();
  return `${Date.now().toString(36)}-${Math.random().toString(36).slice(2, 10)}`;
}

/** 把错误原文收短，去掉接口前缀。 */
function shortReason(msg: string): string {
  const s = msg.replace(/^\/api\/idol:\s*/i, "").trim();
  return s.length > 80 ? `${s.slice(0, 80)}…` : s;
}

/** 把音频块读成去掉前缀的 base64。 */
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

/** phoneFrom 原样交回是否走电话音质。 */
export function phoneFrom(flag: boolean) {
  return flag;
}
