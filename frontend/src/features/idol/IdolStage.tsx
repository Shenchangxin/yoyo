/** 拥有偶像台：名单、当前会话、通话层和侧栏入口。不拥有立绘绘制和通话循环。保存、置顶、发消息、删档和开新会话会改服务端状态。 */
import { useEffect, useRef, useState } from "react";
import { MoreHorizontal, Phone, Pin, PinOff, Plus } from "lucide-react";
import { toast } from "sonner";
import { Button } from "../../components/ui/button";
import { EmptyState } from "../../components/ui/empty-state";
import {
  DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuSeparator, DropdownMenuTrigger,
} from "../../components/ui/dropdown-menu";
import * as api from "../../lib/client";
import { useCopy } from "../../lib/i18n";
import { asArray, str } from "../../lib/normalize";
import { useUI } from "../../lib/store";
import { cn } from "../../lib/utils";
import { PresenceStamp } from "../presence/PresenceRuntime";
import { CallBar, CallSurface } from "./IdolCall";
import { IdolChat } from "./IdolChat";
import { IdolPanel, type IdolPanelKind } from "./IdolPanels";
import {
  affinityOf, blankIdol, groupIdols, historyOf, idolError, idolOf, imageURL, isCooling, linesOf, snapOf, whenLabel,
  type History, type Idol, type Snap,
} from "./model";
import { warmStand } from "./stand/StandView";
import { primeCallAudio, useIdolCall } from "./useIdolCall";

const emptyHistory: History = { current: null, past: [] };

/** 按肖像编号取回图片地址，并缓存在这一轮里。idols 做映射。 */
function usePortraitURLs(idols: Idol[]) {
  const key = idols.map((i) => i.portrait).filter(Boolean).sort().join("|");
  const cache = useRef(new Map<string, string>());
  const [ready, setReady] = useState(0);
  useEffect(() => {
    const missing = (key ? key.split("|") : []).filter((id) => id && !cache.current.has(id));
    if (!missing.length) return;
    let stop = false;
    void Promise.all(missing.map(async (id) => {
      try {
        const o = await api.idolCall("blob", { id });
        const data = str(o?.data);
        if (data) cache.current.set(id, imageURL(data));
      } catch { /* 缺的脸保持底色 */ }
    })).then(() => { if (!stop) setReady((n) => n + 1); });
    return () => { stop = true; };
  }, [key]);
  void ready;
  return cache.current;
}

/** 画名单上的圆脸、桌宠印或色块。props 读取 size、src、idol。 */
function IdolMark(props: { idol: Idol; src: string; size?: number }) {
  const size = props.size ?? 36;
  if (props.src) {
    return (
      <span className="shrink-0 overflow-hidden rounded-full" style={{ width: size, height: size, background: props.idol.color }}>
        <img alt="" className="size-full object-cover" src={props.src} />
      </span>
    );
  }
  if (props.idol.kind === "pet") return <PresenceStamp size={size} className="shrink-0" />;
  return <span className="shrink-0 rounded-full" style={{ width: size, height: size, background: props.idol.color || "#c47b8a" }} />;
}

/** IdolStage 管名单、会话、通话层和侧栏，并把保存与发送写到服务端。 */
export function IdolStage() {
  const copy = useCopy();
  const c = copy.idol;
  const openSettings = useUI((s) => s.openSettings);
  const [snap, setSnap] = useState<Snap | null>(null);
  const [sel, setSel] = useState("");
  const [panel, setPanel] = useState<IdolPanelKind | null>(null);
  const [draft, setDraft] = useState<Idol>(blankIdol());
  const [history, setHistory] = useState<History>(emptyHistory);
  const [viewing, setViewing] = useState("");
  const [text, setText] = useState("");
  const [work, setWork] = useState(false);
  const [sending, setSending] = useState(false);
  const [pending, setPending] = useState("");
  const [saving, setSaving] = useState(false);
  const [notice, setNotice] = useState("");
  const [bootError, setBootError] = useState("");
  const [booted, setBooted] = useState(false);
  const [deskBusy, setDeskBusy] = useState(false);
  const [callOn, setCallOn] = useState(false);
  const [callOpen, setCallOpen] = useState(true);
  const [callWork, setCallWork] = useState(false);
  const [face, setFace] = useState("");
  const [giftNames, setGiftNames] = useState<Record<string, string>>({});
  const [talkFor, setTalkFor] = useState("");
  const talkGen = useRef(0);
  const faces = usePortraitURLs(snap?.idols || []);

  const current = snap?.idols.find((i) => i.id === sel) || null;
  useEffect(() => {
    warmStand(current?.outfit_id || "");
  }, [current?.outfit_id]);
  const aff = affinityOf(snap?.affinities || [], sel);
  const cooling = isCooling(aff.cooldown_until) ? whenLabel(aff.cooldown_until) : "";
  const viewed = history.past.find((p) => p.id === viewing) || null;
  const lines = linesOf(viewed ? viewed.records : history.current?.records || []);
  const ended = !!history.current?.ended && !viewed;
  const blocked = !!cooling || !!history.current?.ended || !!viewed;

  const call = useIdolCall({
    active: callOn,
    idolId: sel,
    name: current?.name,
    color: current?.color,
    portrait: face,
    kind: current?.kind,
    stand: current?.outfit_id,
    work: callWork,
    textOnly: !!snap?.textOnly,
    phone: !!snap?.phone,
    /** 通话结束后关掉通话层，并刷新这段会话。 */
    onEnded: () => {
      setCallOn(false);
      void refreshTalk(sel);
    },
  });

  /** 拉取快照并保住仍然存在的选中项。 */
  async function load() {
    const next = snapOf(await api.idolCall("snapshot"));
    setSnap(next);
    setSel((cur) => (cur && next.idols.some((i) => i.id === cur) ? cur : next.pinnedId || next.idols.find((i) => !i.hidden)?.id || ""));
    return next;
  }

  /** 重拉这一位的会话和快照，过期的请求丢掉。idolId 为假时走另一支，传给 setTalkFor，写入字段 idol_id。 */
  async function refreshTalk(idolId = sel) {
    const gen = ++talkGen.current;
    if (!idolId) {
      setHistory(emptyHistory);
      setTalkFor("");
      return;
    }
    const hist = historyOf(await api.idolCall("history", { idol_id: idolId }));
    if (gen !== talkGen.current) return;
    setHistory(hist);
    setTalkFor(idolId);
    const fresh = snapOf(await api.idolCall("snapshot"));
    if (gen !== talkGen.current) return;
    setSnap(fresh);
  }

  useEffect(() => {
    void load()
      .then(() => setBooted(true))
      .catch((e) => {
        setBootError(idolError(e, c));
        setBooted(true);
      });
    void api.idolCall("gifts").then((g) => {
      const map: Record<string, string> = {};
      for (const item of asArray(g?.catalog)) map[str(item.id)] = str(item.name);
      setGiftNames(map);
    }).catch(() => {});
    // 只在第一次打开时拉快照。
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  useEffect(() => {
    if (!sel) {
      setHistory(emptyHistory);
      setTalkFor("");
      return;
    }
    const gen = ++talkGen.current;
    let stop = false;
    setViewing("");
    setNotice("");
    setTalkFor("");
    void api.idolCall("history", { idol_id: sel }).then((h) => {
      if (stop || gen !== talkGen.current) return;
      setHistory(historyOf(h));
      setTalkFor(sel);
    }).catch((e) => { if (!stop && gen === talkGen.current) setNotice(idolError(e, c)); });
    return () => { stop = true; };
  }, [sel]);

  useEffect(() => {
    const idol = snap?.idols.find((i) => i.id === sel);
    if (!idol?.portrait) { setFace(""); return; }
    if (snap?.pinnedId === idol.id && snap.portrait) { setFace(snap.portrait); return; }
    let stop = false;
    void api.idolCall("blob", { id: idol.portrait }).then((o) => {
      const data = str(o?.data);
      if (!stop) setFace(data);
    }).catch(() => {});
    return () => { stop = true; };
  }, [sel, snap?.pinnedId, snap?.portrait, current?.portrait]);

  /** 保存偶像并刷新名单。next 读取 name，传给 api.idolCall。keepPanel 为假时走另一支。 */
  async function save(next: Idol, keepPanel = false) {
    if (!next.name.trim()) {
      setNotice(c.requiredName);
      return;
    }
    setSaving(true);
    try {
      const saved = idolOf(await api.idolCall("save_idol", next));
      await load();
      setSel(saved.id);
      if (!keepPanel) setPanel(null);
      setNotice("");
    } catch (e) {
      setNotice(idolError(e, c));
    } finally {
      setSaving(false);
    }
  }

  /** 把输入送成一个回合，失败时把文字放回输入框。 */
  async function send() {
    const body = text.trim();
    if (!body || !sel || sending || blocked) return;
    setText("");
    setPending(body);
    setSending(true);
    setNotice("");
    try {
      const res = await api.idolCall("turn", { idol_id: sel, text: body, work });
      if (res?.work_session) toast.success(c.openWork);
    } catch (e) {
      setText(body);
      setNotice(idolError(e, c));
      setPending("");
      setSending(false);
      return;
    }
    setPending("");
    setSending(false);
    try {
      await refreshTalk(sel);
    } catch (e) {
      setNotice(idolError(e, c));
    }
  }

  /** 把这一位放到桌面，或从桌面取下。id 为假时用 sel，与 i.id 比较是否相等。 */
  async function place(id?: string) {
    const idol = (snap?.idols || []).find((i) => i.id === (id || sel));
    if (!idol) return;
    setDeskBusy(true);
    setNotice("");
    try {
      if (idol.pinned) {
        await api.idolCall("pin", { id: "" });
        toast.success(c.takenOff);
      } else {
        await api.placeIdol(idol.id);
        toast.success(c.placed);
      }
      await load();
    } catch (e) {
      const msg = idolError(e, c);
      setNotice(msg);
      toast.error(msg);
    } finally {
      setDeskBusy(false);
    }
  }

  /** 开始或回到这一位的通话；被冷却、结束或只读挡住时只留下提示。id 为假时用 sel。 */
  function startCall(id?: string) {
    const target = id || sel;
    const idol = (snap?.idols || []).find((i) => i.id === target);
    if (!idol) return;
    if (callOn && target === sel) {
      setCallOpen(true);
      return;
    }
    if (!id && blocked) {
      setNotice(cooling ? c.cooldown : ended ? c.ended : c.readOnly);
      return;
    }
    if (target !== sel) setSel(target);
    primeCallAudio();
    setPanel(null);
    setCallWork(work && idol.work_enabled && snap?.work !== false);
    setCallOpen(true);
    setCallOn(true);
  }

  /** 打开编辑，没有对象就用空白草稿。next 为真时才继续，展开传递。 */
  function openEdit(next?: Idol) {
    setDraft(next ? { ...next } : blankIdol());
    setPanel("edit");
  }

  const panelIdol = panel === "edit" && !draft.id ? null : current;
  const groups = groupIdols(snap?.idols || [], snap?.circles || []);
  const names = Object.fromEntries((snap?.idols || []).map((i) => [i.id, i.name]));
  const tier = (c.tier as Record<string, string>)[aff.tier] || aff.tier;

  return (
    <div className="flex h-full min-h-0" data-testid="idol-stage">
      <aside className="flex w-60 shrink-0 flex-col border-r border-border">
        <div className="px-3 pb-1 pt-3 text-[13px] font-medium">{c.title}</div>
        <p className="px-3 pb-2 text-[11px] leading-relaxed text-muted">{c.railHint}</p>
        <div className="min-h-0 flex-1 overflow-auto px-2 pb-2">
          {groups.map((g) => (
            <div key={g.id || "loose"} className="mb-2">
              {(g.name || groups.length > 1) ? (
                <div className="px-2 pb-1 pt-2 text-[11px] text-muted">{g.name || c.unassigned}</div>
              ) : null}
              {g.idols.map((i) => (
                <div
                  key={i.id}
                  className={cn("group flex items-center rounded-lg pr-1 hover:bg-lift", sel === i.id && "bg-lift")}
                >
                  <button
                    type="button"
                    className="flex min-w-0 flex-1 items-center gap-2 px-2 py-1.5 text-left"
                    onClick={() => { setSel(i.id); setPanel(null); }}
                  >
                    <IdolMark idol={i} src={i.portrait ? faces.get(i.portrait) || "" : ""} />
                    <span className="min-w-0 flex-1">
                      <span className="flex items-center gap-1">
                        <span className="truncate text-[13px]">{i.name}</span>
                        {i.pinned ? <span className="shrink-0 text-[10px] text-muted">{c.onDesktop}</span> : null}
                      </span>
                      {i.intro ? <span className="block truncate text-[11px] text-muted">{i.intro}</span> : null}
                    </span>
                  </button>
                  <div className="flex shrink-0 items-center gap-0.5">
                    <button
                      type="button"
                      className="grid size-7 place-items-center rounded-full text-muted transition duration-200 hover:bg-background hover:text-foreground"
                      title={c.call}
                      aria-label={c.call}
                      onClick={() => startCall(i.id)}
                    >
                      <Phone className="size-3.5" />
                    </button>
                    <button
                      type="button"
                      className="grid size-7 place-items-center rounded-full text-muted transition duration-200 hover:bg-background hover:text-foreground disabled:opacity-40"
                      title={i.pinned ? c.unpin : c.pin}
                      aria-label={i.pinned ? c.unpin : c.pin}
                      disabled={deskBusy}
                      onClick={() => void place(i.id)}
                    >
                      {i.pinned ? <PinOff className="size-3.5" /> : <Pin className="size-3.5" />}
                    </button>
                  </div>
                </div>
              ))}
            </div>
          ))}
          {booted && !bootError && groups.length === 0 ? <p className="px-2 text-[12px] text-muted">{c.empty}</p> : null}
        </div>
        <div className="flex gap-1 border-t border-border p-2">
          <Button size="sm" variant="lift" className="flex-1" onClick={() => openEdit()}>
            <Plus className="size-3.5" />{c.create}
          </Button>
          <Button size="sm" variant="ghost" onClick={() => setPanel("circles")}>{c.circle}</Button>
        </div>
      </aside>
      <section className="relative flex min-w-0 flex-1 flex-col">
        {!booted ? <div className="grid flex-1 place-items-center text-[13px] text-muted">{c.loading}</div> : null}
        {booted && bootError ? (
          <div className="grid flex-1 place-items-center">
            <EmptyState title={c.callFailed} body={bootError} action={<Button size="sm" onClick={() => { setBootError(""); void load().catch((e) => setBootError(idolError(e, c))); }}>{c.retry}</Button>} />
          </div>
        ) : null}
        {booted && !bootError && current ? (
          <>
            <header className="flex flex-wrap items-center gap-2 border-b border-border px-4 py-3">
              <IdolMark idol={current} src={current.portrait ? faces.get(current.portrait) || "" : ""} size={32} />
              <div className="min-w-0 flex-1">
                <div className="flex items-center gap-2">
                  <h1 className="truncate text-[15px] font-medium">{current.name}</h1>
                  <span className="shrink-0 text-[12px] text-muted">{tier} · {aff.score}</span>
                  {current.pinned ? <span className="shrink-0 rounded-full bg-lift px-2 py-0.5 text-[10px] text-muted">{c.onDesktop}</span> : null}
                </div>
                <p className="truncate text-[12px] text-muted">{viewed?.title || history.current?.title || current.intro}</p>
              </div>
              <Button size="sm" disabled={!callOn && blocked} title={callOn ? c.backToCall : cooling ? c.cooldown : viewed ? c.readOnly : ended ? c.ended : c.call} onClick={() => startCall()}>
                <Phone className="size-3.5" />{callOn ? c.backToCall : c.call}
              </Button>
              <Button size="sm" variant="lift" disabled={deskBusy} title={c.placeHint} onClick={() => void place()}>
                {deskBusy ? c.placing : current.pinned ? c.unpin : c.pin}
              </Button>
              <DropdownMenu>
                <DropdownMenuTrigger asChild>
                  <Button size="icon" variant="ghost" aria-label={c.more}><MoreHorizontal /></Button>
                </DropdownMenuTrigger>
                <DropdownMenuContent align="end">
                  <DropdownMenuItem onSelect={() => void api.idolCall("new_conversation", { idol_id: current.id }).then(() => { setViewing(""); return refreshTalk(current.id); }).then(() => toast.success(c.newChatDone)).catch((e) => setNotice(idolError(e, c)))}>{c.newChat}</DropdownMenuItem>
                  <DropdownMenuItem onSelect={() => setPanel("history")}>{c.pastTitle}</DropdownMenuItem>
                  <DropdownMenuItem onSelect={() => openEdit(current)}>{c.edit}</DropdownMenuItem>
                  <DropdownMenuSeparator />
                  <DropdownMenuItem onSelect={() => setPanel("memories")}>{c.memories}</DropdownMenuItem>
                  <DropdownMenuItem onSelect={() => setPanel("gifts")}>{c.gifts}</DropdownMenuItem>
                  <DropdownMenuItem onSelect={() => setPanel("lore")}>{c.lore}</DropdownMenuItem>
                  <DropdownMenuItem onSelect={() => setPanel("stats")}>{c.stats}</DropdownMenuItem>
                  <DropdownMenuItem onSelect={() => setPanel("circles")}>{c.circlesTitle}</DropdownMenuItem>
                </DropdownMenuContent>
              </DropdownMenu>
            </header>
            {snap && !snap.chatReady ? (
              <div className="flex items-center gap-2 border-b border-warning/30 bg-warning/10 px-4 py-2 text-[12px] text-warning">
                <span className="min-w-0 flex-1">{c.modelBanner}</span>
                <Button size="sm" variant="lift" onClick={() => openSettings("idol")}>{c.openSettings}</Button>
              </div>
            ) : null}
            {callOn && !callOpen ? (
              <CallBar
                name={current.name}
                elapsed={call.elapsed}
                dialing={call.phase === "dialing"}
                onExpand={() => setCallOpen(true)}
                onHangup={call.hangup}
              />
            ) : null}
            <div className="relative flex min-h-0 flex-1 flex-col">
              {callOn && callOpen ? (
                <div className="absolute inset-0 z-30">
                  <CallSurface
                    who={call.who.name ? call.who : { name: current.name, color: current.color, portrait: face, kind: current.kind, stand: current.outfit_id }}
                    phase={call.phase}
                    emotion={call.emotion}
                    caption={call.caption}
                    notice={call.notice}
                    elapsed={call.elapsed}
                    muteMic={call.muteMic}
                    muteSpk={call.muteSpk}
                    textOnly={call.textOnly}
                    draft={call.draft}
                    work={callWork}
                    showWork={current.work_enabled && snap?.work !== false}
                    onWork={setCallWork}
                    onMuteMic={() => call.setMuteMic((v) => !v)}
                    onMuteSpk={() => call.setMuteSpk((v) => !v)}
                    onDraft={call.setDraft}
                    onSend={call.sendDraft}
                    onHangup={call.hangup}
                    onCollapse={() => { setCallOpen(false); void refreshTalk(sel); }}
                  />
                </div>
              ) : (
                <IdolChat
                  idol={current}
                  lines={lines}
                  pending={pending}
                  sending={sending}
                  notice={notice}
                  text={text}
                  work={work}
                  canWork={current.work_enabled && snap?.work !== false}
                  ended={ended}
                  cooling={viewed ? "" : cooling}
                  viewingTitle={viewed ? (viewed.title || whenLabel(viewed.created_at)) : ""}
                  ready={talkFor === sel}
                  giftNames={giftNames}
                  onText={setText}
                  onWork={setWork}
                  onSend={() => void send()}
                  onNewChat={() => void api.idolCall("new_conversation", { idol_id: current.id }).then(() => { setViewing(""); return refreshTalk(current.id); }).catch((e) => setNotice(idolError(e, c)))}
                  onOpenGifts={() => setPanel("gifts")}
                  onBack={() => setViewing("")}
                  onOpenWork={(id) => void api.raiseSession(id)}
                />
              )}
              {panel && !(callOn && callOpen) ? (
                <IdolPanel
                  kind={panel}
                  idol={panelIdol}
                  idols={snap?.idols || []}
                  circles={snap?.circles || []}
                  lore={snap?.lore || []}
                  draft={draft}
                  history={history}
                  viewing={viewing}
                  names={names}
                  saving={saving}
                  onDraft={setDraft}
                  onClose={() => setPanel(null)}
                  onSave={(i) => void save(i, panel !== "edit")}
                  onDelete={(id) => void api.idolCall("delete_idol", { id }).then(async () => { setPanel(null); setSel(""); await load(); }).catch((e) => setNotice(idolError(e, c)))}
                  onView={setViewing}
                  onDeleteChat={(id) => void api.idolCall("delete_conversation", { id }).then(() => { if (viewing === id) setViewing(""); return refreshTalk(current.id); }).catch((e) => toast.error(idolError(e, c)))}
                  onChanged={() => void refreshTalk(current.id)}
                />
              ) : null}
            </div>
          </>
        ) : null}
        {booted && !bootError && !current ? (
          <div className="relative grid flex-1 place-items-center">
            <EmptyState
              title={c.empty}
              body={c.railHint}
              action={<Button size="sm" onClick={() => openEdit()}><Plus className="size-3.5" />{c.create}</Button>}
            />
            {panel ? (
              <IdolPanel
                kind={panel}
                idol={null}
                idols={snap?.idols || []}
                circles={snap?.circles || []}
                lore={snap?.lore || []}
                draft={draft}
                history={history}
                viewing=""
                names={names}
                saving={saving}
                onDraft={setDraft}
                onClose={() => setPanel(null)}
                onSave={(i) => void save(i)}
                onDelete={() => {}}
                onView={() => {}}
                onDeleteChat={() => {}}
                onChanged={() => void load()}
              />
            ) : null}
          </div>
        ) : null}
      </section>
    </div>
  );
}
