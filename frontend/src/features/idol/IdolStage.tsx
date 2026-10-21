import { useEffect, useState } from "react";
import { toast } from "sonner";
import { Button } from "../../components/ui/button";
import { Input } from "../../components/ui/input";
import * as api from "../../lib/client";
import { useCopy } from "../../lib/i18n";
import { asArray, str } from "../../lib/normalize";
import { cn } from "../../lib/utils";

type Idol = {
  id: string;
  name: string;
  intro: string;
  color: string;
  spoken_lang: string;
  subtitle_lang: string;
  sheet: string;
  facts: string;
  readings: string;
  voice_id: string;
  portrait: string;
  circle_id: string;
  work_enabled: boolean;
  work_profile_id: string;
  pinned: boolean;
  hidden: boolean;
  liked_gifts: string[];
};

type Rec = {
  id: string;
  kind: string;
  user?: string;
  segments?: { spoken?: string; subtitle?: string }[];
  work_session?: string;
  gift_id?: string;
  photo_id?: string;
};

const blank = (): Idol => ({
  id: "", name: "", intro: "", color: "#c47b8a", spoken_lang: "zh", subtitle_lang: "zh",
  sheet: "", facts: "", readings: "", voice_id: "", portrait: "", circle_id: "",
  work_enabled: true, work_profile_id: "", pinned: false, hidden: false, liked_gifts: [],
});

export function IdolStage() {
  const copy = useCopy();
  const c = copy.idol;
  const [idols, setIdols] = useState<Idol[]>([]);
  const [circles, setCircles] = useState<{ id: string; name: string }[]>([]);
  const [sel, setSel] = useState("");
  const [draft, setDraft] = useState<Idol | null>(null);
  const [tab, setTab] = useState<"talk" | "edit" | "memories" | "gifts" | "lore" | "stats">("talk");
  const [records, setRecords] = useState<Rec[]>([]);
  const [past, setPast] = useState<{ id: string; title?: string; records?: Rec[] }[]>([]);
  const [text, setText] = useState("");
  const [work, setWork] = useState(false);
  const [busy, setBusy] = useState(false);
  const [tier, setTier] = useState("trust");
  const [memories, setMemories] = useState<any[]>([]);
  const [gifts, setGifts] = useState<any>(null);
  const [lore, setLore] = useState<any[]>([]);
  const [loreTitle, setLoreTitle] = useState("");
  const [loreBody, setLoreBody] = useState("");
  const [stats, setStats] = useState<any>(null);
  const [circleName, setCircleName] = useState("");

  async function load() {
    const snap = await api.idolCall("snapshot");
    const list = asArray(snap?.idols).map(idolOf);
    setIdols(list);
    setCircles(asArray(snap?.circles).map((x: any) => ({ id: str(x.id), name: str(x.name) })));
    setSel((cur) => cur || list.find((i) => i.pinned)?.id || list[0]?.id || "");
  }

  useEffect(() => { void load().catch((e) => toast.error(String(e?.message || e))); }, []);

  useEffect(() => {
    if (!sel) return;
    void refreshTalk();
  }, [sel]);

  async function refreshTalk() {
    const hist = await api.idolCall("history", { idol_id: sel });
    setRecords(asArray(hist?.current?.records));
    setPast(asArray(hist?.past).map((p: any) => ({ id: str(p.id), title: str(p.title), records: asArray(p.records) })));
    const snap = await api.idolCall("snapshot");
    const aff = asArray(snap?.affinities).find((a: any) => str(a.idol_id) === sel);
    if (aff) setTier(str(aff.tier) || "trust");
  }

  function idolOf(v: any): Idol {
    return {
      ...blank(),
      id: str(v.id),
      name: str(v.name),
      intro: str(v.intro),
      color: str(v.color) || "#c47b8a",
      spoken_lang: str(v.spoken_lang) || "zh",
      subtitle_lang: str(v.subtitle_lang) || "zh",
      sheet: str(v.sheet),
      facts: str(v.facts),
      readings: str(v.readings),
      voice_id: str(v.voice_id),
      portrait: str(v.portrait),
      circle_id: str(v.circle_id),
      work_enabled: v.work_enabled !== false,
      work_profile_id: str(v.work_profile_id),
      pinned: !!v.pinned,
      hidden: !!v.hidden,
      liked_gifts: asArray(v.liked_gifts).map((x) => str(x)),
    };
  }

  const current = idols.find((i) => i.id === sel) || null;

  async function save(next: Idol) {
    setBusy(true);
    try {
      const saved = idolOf(await api.idolCall("save_idol", next));
      await load();
      setSel(saved.id);
      setDraft(null);
      setTab("talk");
    } catch (e: any) {
      toast.error(String(e?.message || e));
    } finally {
      setBusy(false);
    }
  }

  async function send() {
    const body = text.trim();
    if (!body || !sel) return;
    setText("");
    setBusy(true);
    try {
      const res = await api.idolCall("turn", { idol_id: sel, text: body, work });
      if (res?.work_session) toast.success(c.openWork);
      await refreshTalk();
    } catch (e: any) {
      const msg = String(e?.message || e);
      toast.error(msg === "cooldown" ? c.cooldown : msg === "ended" ? c.ended : msg);
    } finally {
      setBusy(false);
    }
  }

  async function openTab(next: typeof tab) {
    setTab(next);
    if (!sel) return;
    if (next === "memories") {
      const m = await api.idolCall("memories", { idol_id: sel });
      setMemories(asArray(m?.memories));
    }
    if (next === "gifts") setGifts(await api.idolCall("gifts"));
    if (next === "lore") {
      const all = await api.idolCall("snapshot");
      setLore(asArray(all?.lore).filter((d: any) => str(d.idol_id) === sel));
    }
    if (next === "stats") setStats(await api.idolCall("stats"));
  }

  return (
    <div className="flex h-full min-h-0" data-testid="idol-stage">
      <aside className="flex w-56 shrink-0 flex-col gap-1 overflow-auto border-r border-border p-3">
        <div className="mb-2 text-[13px] font-medium">{c.title}</div>
        {idols.filter((i) => !i.hidden).map((i) => (
          <button
            key={i.id}
            type="button"
            className={cn("flex items-center gap-2 rounded-lg px-2 py-1.5 text-left text-[13px] hover:bg-lift", sel === i.id && "bg-lift")}
            onClick={() => { setSel(i.id); setDraft(null); setTab("talk"); }}
          >
            <span className="size-2.5 rounded-full" style={{ background: i.color }} />
            <span className="min-w-0 flex-1 truncate">{i.name}</span>
            {i.pinned ? <span className="text-[10px] text-muted">●</span> : null}
          </button>
        ))}
        {idols.length === 0 ? <p className="text-[12px] text-muted">{c.empty}</p> : null}
        <Button size="sm" variant="lift" className="mt-2" onClick={() => { setDraft(blank()); setTab("edit"); }}>{c.create}</Button>
        <form className="mt-4 flex gap-1" onSubmit={(e) => {
          e.preventDefault();
          const name = circleName.trim();
          if (!name) return;
          void api.idolCall("save_circle", { name }).then(() => { setCircleName(""); return load(); });
        }}>
          <Input className="h-8 text-[12px]" value={circleName} placeholder={c.circleNew} onChange={(e) => setCircleName(e.target.value)} />
        </form>
      </aside>
      <section className="flex min-w-0 flex-1 flex-col">
        {draft ? (
          <Editor idol={draft} circles={circles} busy={busy} onChange={setDraft} onSave={() => void save(draft)} />
        ) : current ? (
          <>
            <header className="flex flex-wrap items-center gap-2 border-b border-border px-4 py-3">
              <span className="size-3 rounded-full" style={{ background: current.color }} />
              <span className="text-[14px] font-medium">{current.name}</span>
              <span className="text-[12px] text-muted">{(c.tier as any)[tier] || tier}</span>
              <div className="ml-auto flex flex-wrap gap-1">
                <Button size="sm" variant="lift" onClick={() => void api.idolCall("pin", { id: current.pinned ? "" : current.id }).then(load)}>{current.pinned ? c.unpin : c.pin}</Button>
                <Button size="sm" variant="lift" onClick={() => { void api.showCompanion(); void api.setCompanionCall(true); }}>{c.call}</Button>
                <Button size="sm" variant="lift" onClick={() => void api.idolCall("new_conversation", { idol_id: current.id }).then(refreshTalk)}>{c.newChat}</Button>
                <Button size="sm" variant="lift" onClick={() => { setDraft(current); setTab("edit"); }}>{c.edit}</Button>
              </div>
            </header>
            <nav className="flex gap-1 px-4 pt-2 text-[12px]">
              {(["talk", "memories", "gifts", "lore", "stats"] as const).map((k) => (
                <button key={k} type="button" className={cn("rounded-md px-2 py-1 hover:bg-lift", tab === k && "bg-lift")} onClick={() => void openTab(k)}>
                  {(c as any)[k]}
                </button>
              ))}
            </nav>
            {tab === "talk" ? (
              <div className="flex min-h-0 flex-1 flex-col">
                <div className="min-h-0 flex-1 space-y-3 overflow-auto px-4 py-3">
                  {records.map((r) => (
                    <article key={r.id} className="text-[13px]">
                      {r.user ? <p className="text-muted">{r.user}</p> : null}
                      {(r.segments || []).map((s, i) => (
                        <p key={i}>{s.subtitle || s.spoken}</p>
                      ))}
                      {r.work_session ? (
                        <button type="button" className="text-[12px] text-muted underline" onClick={() => void api.raiseSession(r.work_session!)}>{c.openWork}</button>
                      ) : null}
                    </article>
                  ))}
                  {past.length ? (
                    <div className="pt-4 text-[12px] text-muted">
                      {c.history}
                      {past.map((p) => (
                        <button key={p.id} type="button" className="block text-left underline" onClick={() => setRecords(p.records || [])}>{p.title || p.id}</button>
                      ))}
                    </div>
                  ) : null}
                </div>
                <form className="border-t border-border p-3" onSubmit={(e) => { e.preventDefault(); void send(); }}>
                  <label className="mb-2 flex items-center gap-2 text-[12px] text-muted">
                    <input type="checkbox" checked={work} onChange={(e) => setWork(e.target.checked)} />
                    {c.work}
                  </label>
                  <div className="flex gap-2">
                    <Input value={text} onChange={(e) => setText(e.target.value)} placeholder={c.talk} />
                    <Button type="submit" size="sm" disabled={busy}>{c.send}</Button>
                  </div>
                </form>
              </div>
            ) : null}
            {tab === "memories" ? (
              <ul className="space-y-2 overflow-auto p-4 text-[13px]">
                {memories.map((m) => (
                  <li key={str(m.id)} className="flex items-start justify-between gap-2">
                    <span>{str(m.text)}{m.superseded_by ? " ·" : ""}</span>
                    <button type="button" className="text-[12px] text-muted" onClick={() => void api.idolCall("delete_memory", { id: m.id }).then(() => openTab("memories"))}>{c.deleteMemory}</button>
                  </li>
                ))}
              </ul>
            ) : null}
            {tab === "gifts" ? (
              <div className="space-y-3 overflow-auto p-4 text-[13px]">
                <div className="flex items-center gap-2">
                  <span>{c.draws}: {gifts?.draws ?? 0}</span>
                  <Button size="sm" variant="lift" onClick={() => void api.idolCall("draw").then(() => openTab("gifts")).catch((e) => toast.error(String(e?.message || e)))}>{c.draw}</Button>
                  <Button size="sm" variant="lift" onClick={() => void api.idolCall("reset").then(() => openTab("gifts")).catch((e) => toast.error(String(e?.message || e)))}>{c.reset}</Button>
                </div>
                {asArray(gifts?.catalog).map((g: any) => {
                  const held = asArray(gifts?.inventory).find((s: any) => s.gift_id === g.id)?.count || 0;
                  return (
                    <div key={g.id} className="flex items-center justify-between gap-2">
                      <span>{g.name} · {g.rarity} · {held}</span>
                      <Button size="sm" variant="lift" disabled={!held} onClick={() => void api.idolCall("give", { idol_id: sel, gift_id: g.id }).then(refreshTalk)}>{c.give}</Button>
                    </div>
                  );
                })}
              </div>
            ) : null}
            {tab === "lore" ? (
              <div className="space-y-3 overflow-auto p-4">
                {lore.map((d) => (
                  <article key={str(d.id)}>
                    <div className="text-[13px] font-medium">{str(d.title)}</div>
                    <p className="whitespace-pre-wrap text-[12px] text-muted">{str(d.body)}</p>
                    <button type="button" className="text-[12px] text-muted" onClick={() => void api.idolCall("delete_lore", { id: d.id }).then(() => openTab("lore"))}>{c.delete}</button>
                  </article>
                ))}
                <Input value={loreTitle} placeholder={c.loreTitle} onChange={(e) => setLoreTitle(e.target.value)} />
                <textarea className="min-h-24 w-full rounded-md border border-border bg-background p-2 text-[13px]" value={loreBody} placeholder={c.loreBody} onChange={(e) => setLoreBody(e.target.value)} />
                <Button size="sm" onClick={() => void api.idolCall("save_lore", { idol_id: sel, title: loreTitle, body: loreBody }).then(() => { setLoreTitle(""); setLoreBody(""); return openTab("lore"); })}>{c.addLore}</Button>
              </div>
            ) : null}
            {tab === "stats" && stats ? (
              <div className="space-y-2 p-4 text-[13px]">
                <div>{c.turns}: {stats.turns}</div>
                <div>{c.callTime}: {stats.call_sec}s · {stats.calls}</div>
                <div>{c.giftsGiven}: {stats.gifts}</div>
                <div>{c.memories}: {stats.memories}</div>
                <div>{c.draws}: {stats.draws_left}</div>
                <div>{c.achievements}: {asArray(stats.achievements).join(", ") || "—"}</div>
                <div>{c.tasks}: {stats.tasks?.talk ? c.taskTalk : ""} {stats.tasks?.call ? c.taskCall : ""} {stats.tasks?.gift ? c.taskGift : ""}</div>
              </div>
            ) : null}
          </>
        ) : (
          <div className="grid flex-1 place-items-center text-[13px] text-muted">{c.empty}</div>
        )}
      </section>
    </div>
  );
}

function Editor(props: {
  idol: Idol;
  circles: { id: string; name: string }[];
  busy: boolean;
  onChange: (i: Idol) => void;
  onSave: () => void;
}) {
  const c = useCopy().idol;
  const i = props.idol;
  const set = (patch: Partial<Idol>) => props.onChange({ ...i, ...patch });
  return (
    <div className="mx-auto flex w-full max-w-xl flex-col gap-2 overflow-auto p-4">
      <Field label={c.name} value={i.name} onChange={(name) => set({ name })} />
      <Field label={c.intro} value={i.intro} onChange={(intro) => set({ intro })} />
      <Area label={c.sheet} value={i.sheet} onChange={(sheet) => set({ sheet })} />
      <Area label={c.facts} value={i.facts} onChange={(facts) => set({ facts })} />
      <Area label={c.readings} value={i.readings} onChange={(readings) => set({ readings })} />
      <div className="flex gap-2">
        <Field label={c.spoken} value={i.spoken_lang} onChange={(spoken_lang) => set({ spoken_lang })} />
        <Field label={c.subtitle} value={i.subtitle_lang} onChange={(subtitle_lang) => set({ subtitle_lang })} />
      </div>
      <div className="flex gap-2">
        <Field label={c.color} value={i.color} onChange={(color) => set({ color })} />
        <Field label={c.voice} value={i.voice_id} onChange={(voice_id) => set({ voice_id })} />
      </div>
      <label className="text-[12px] text-muted">
        {c.circle}
        <select className="mt-1 h-8 w-full rounded-md border border-border bg-background px-2 text-[13px]" value={i.circle_id} onChange={(e) => set({ circle_id: e.target.value })}>
          <option value="">{c.noCircle}</option>
          {props.circles.map((cir) => <option key={cir.id} value={cir.id}>{cir.name}</option>)}
        </select>
      </label>
      <label className="flex items-center gap-2 text-[12px] text-muted">
        <input type="checkbox" checked={i.work_enabled} onChange={(e) => set({ work_enabled: e.target.checked })} />
        {c.work}
      </label>
      <label className="text-[12px] text-muted">
        {c.portrait}
        <input className="mt-1 block text-[12px]" type="file" accept="image/*" onChange={(e) => {
          const file = e.target.files?.[0];
          if (!file) return;
          const r = new FileReader();
          r.onload = () => {
            void api.idolCall("save_blob", { data: String(r.result || "") }).then((out) => set({ portrait: str(out?.id) }));
          };
          r.readAsDataURL(file);
        }} />
      </label>
      <Button size="sm" disabled={props.busy} onClick={props.onSave}>{c.save}</Button>
    </div>
  );
}

function Field(props: { label: string; value: string; onChange: (v: string) => void }) {
  return (
    <label className="block text-[12px] text-muted">
      {props.label}
      <Input className="mt-1 h-8" value={props.value} onChange={(e) => props.onChange(e.target.value)} />
    </label>
  );
}

function Area(props: { label: string; value: string; onChange: (v: string) => void }) {
  return (
    <label className="block text-[12px] text-muted">
      {props.label}
      <textarea className="mt-1 min-h-20 w-full rounded-md border border-border bg-background p-2 text-[13px]" value={props.value} onChange={(e) => props.onChange(e.target.value)} />
    </label>
  );
}
