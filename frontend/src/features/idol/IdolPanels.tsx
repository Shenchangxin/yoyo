import { useEffect, useState } from "react";
import { X } from "lucide-react";
import { toast } from "sonner";
import { Button } from "../../components/ui/button";
import { Input } from "../../components/ui/input";
import * as api from "../../lib/client";
import { useCopy } from "../../lib/i18n";
import { asArray, num, str } from "../../lib/normalize";
import {
  achievementLabel,
  blankIdol,
  clock,
  idolError,
  imageURL,
  whenLabel,
  type Circle,
  type Conversation,
  type Idol,
  type Lore,
} from "./model";

export type IdolPanelKind = "edit" | "memories" | "gifts" | "lore" | "stats" | "circles" | "history";

const titles: Record<IdolPanelKind, keyof ReturnType<typeof useCopy>["idol"]> = {
  edit: "edit",
  memories: "memories",
  gifts: "gifts",
  lore: "lore",
  stats: "stats",
  circles: "circlesTitle",
  history: "pastTitle",
};

export function IdolPanel(props: {
  kind: IdolPanelKind;
  idol: Idol | null;
  idols: Idol[];
  circles: Circle[];
  lore: Lore[];
  draft: Idol;
  history: { current: Conversation | null; past: Conversation[] };
  viewing: string;
  names: Record<string, string>;
  onDraft: (i: Idol) => void;
  onClose: () => void;
  onSave: (i: Idol) => void;
  onDelete: (id: string) => void;
  onView: (id: string) => void;
  onDeleteChat: (id: string) => void;
  onChanged: () => void;
  saving: boolean;
}) {
  const c = useCopy().idol;
  const title = c[titles[props.kind]] as string;
  return (
    <aside className="absolute inset-y-0 right-0 z-20 flex w-[min(420px,100%)] flex-col border-l border-border bg-background shadow-[var(--shadow-popover)]">
      <header className="flex items-center gap-2 border-b border-border px-4 py-3">
        <h2 className="min-w-0 flex-1 truncate text-[14px] font-medium">{title}</h2>
        <button type="button" className="grid size-8 place-items-center rounded-lg text-muted hover:bg-lift" aria-label={c.close} onClick={props.onClose}>
          <X className="size-4" />
        </button>
      </header>
      <div className="min-h-0 flex-1 overflow-auto p-4">
        {props.kind === "edit" ? <Editor {...props} /> : null}
        {props.kind === "memories" ? <Memories idol={props.idol} names={props.names} onChanged={props.onChanged} /> : null}
        {props.kind === "gifts" ? <Gifts idol={props.idol} onChanged={props.onChanged} onSave={props.onSave} /> : null}
        {props.kind === "lore" ? <LorePanel idol={props.idol} lore={props.lore} onChanged={props.onChanged} /> : null}
        {props.kind === "stats" ? <Stats names={props.names} idol={props.idol} /> : null}
        {props.kind === "circles" ? <Circles {...props} /> : null}
        {props.kind === "history" ? <HistoryList {...props} /> : null}
      </div>
    </aside>
  );
}

function Editor(props: {
  idol: Idol | null;
  circles: Circle[];
  draft: Idol;
  onDraft: (i: Idol) => void;
  onSave: (i: Idol) => void;
  onDelete: (id: string) => void;
  saving: boolean;
}) {
  const c = useCopy().idol;
  const i = props.draft;
  const set = (patch: Partial<Idol>) => props.onDraft({ ...i, ...patch });
  const [profiles, setProfiles] = useState<{ id: string; name: string }[]>([]);
  const [preview, setPreview] = useState("");
  const [askDelete, setAskDelete] = useState(false);
  useEffect(() => {
    void api.idolCall("profiles").then((rows) => {
      setProfiles(asArray(rows).map((p: any) => ({ id: str(p.id), name: str(p.name) })).filter((p) => p.id));
    }).catch(() => {});
  }, []);
  useEffect(() => {
    if (!i.portrait) { setPreview(""); return; }
    let stop = false;
    void api.idolCall("blob", { id: i.portrait }).then((o) => {
      const data = str(o?.data);
      if (!stop && data) setPreview(imageURL(data));
    }).catch(() => {});
    return () => { stop = true; };
  }, [i.portrait]);
  return (
    <div className="flex flex-col gap-3">
      <p className="text-[12px] leading-relaxed text-muted">{c.editorHint}</p>
      <Field label={c.name} value={i.name} onChange={(name) => set({ name })} />
      <Field label={c.intro} value={i.intro} onChange={(intro) => set({ intro })} />
      <Area label={c.sheet} value={i.sheet} onChange={(sheet) => set({ sheet })} />
      <Area label={c.facts} value={i.facts} onChange={(facts) => set({ facts })} />
      <Area label={c.readings} value={i.readings} onChange={(readings) => set({ readings })} />
      <div className="grid grid-cols-2 gap-2">
        <Lang label={c.spoken} value={i.spoken_lang} onChange={(spoken_lang) => set({ spoken_lang })} />
        <Lang label={c.subtitle} value={i.subtitle_lang} onChange={(subtitle_lang) => set({ subtitle_lang })} />
      </div>
      <label className="block text-[12px] text-muted">
        {c.color}
        <span className="mt-1 flex items-center gap-2">
          <input type="color" value={/^#[0-9a-fA-F]{6}$/.test(i.color) ? i.color : "#c47b8a"} onChange={(e) => set({ color: e.target.value })} />
          <Input className="h-8" value={i.color} onChange={(e) => set({ color: e.target.value })} />
        </span>
      </label>
      <Field label={c.voice} value={i.voice_id} onChange={(voice_id) => set({ voice_id })} />
      <label className="block text-[12px] text-muted">
        {c.circle}
        <select className="mt-1 h-8 w-full rounded-md border border-border bg-background px-2 text-[13px]" value={i.circle_id} onChange={(e) => set({ circle_id: e.target.value })}>
          <option value="">{c.noCircle}</option>
          {props.circles.map((cir) => <option key={cir.id} value={cir.id}>{cir.name}</option>)}
        </select>
      </label>
      <label className="block text-[12px] text-muted">
        {c.workProfile}
        <select className="mt-1 h-8 w-full rounded-md border border-border bg-background px-2 text-[13px]" value={i.work_profile_id} onChange={(e) => set({ work_profile_id: e.target.value })}>
          <option value="">{c.profileDefault}</option>
          {profiles.map((p) => <option key={p.id} value={p.id}>{p.name}</option>)}
        </select>
      </label>
      <label className="flex items-center gap-2 text-[12px] text-muted">
        <input type="checkbox" checked={i.work_enabled} onChange={(e) => set({ work_enabled: e.target.checked })} />
        {c.work}
      </label>
      <label className="flex items-center gap-2 text-[12px] text-muted">
        <input type="checkbox" checked={i.hidden} onChange={(e) => set({ hidden: e.target.checked })} />
        {c.hidden}
      </label>
      <label className="text-[12px] text-muted">
        {c.portrait}
        {preview ? <img alt="" src={preview} className="mt-2 size-20 rounded-full object-cover" /> : null}
        <input className="mt-1 block text-[12px]" type="file" accept="image/*" onChange={(e) => {
          const file = e.target.files?.[0];
          if (!file) return;
          const r = new FileReader();
          r.onload = () => {
            void api.idolCall("save_blob", { data: String(r.result || "") }).then((out) => set({ portrait: str(out?.id) })).catch((err) => toast.error(idolError(err, c)));
          };
          r.readAsDataURL(file);
        }} />
      </label>
      <Button disabled={props.saving || !i.name.trim()} onClick={() => props.onSave(i)}>{c.save}</Button>
      {props.idol?.id ? (
        askDelete ? (
          <div className="rounded-xl border border-border p-3 text-[12px]">
            <p className="text-muted">{c.deleteIdolBody}</p>
            <div className="mt-2 flex gap-2">
              <Button size="sm" variant="danger" onClick={() => props.onDelete(props.idol!.id)}>{c.deleteIdol}</Button>
              <Button size="sm" variant="lift" onClick={() => setAskDelete(false)}>{c.cancel}</Button>
            </div>
          </div>
        ) : <Button variant="danger" onClick={() => setAskDelete(true)}>{c.deleteIdol}</Button>
      ) : null}
    </div>
  );
}

function Memories(props: { idol: Idol | null; names: Record<string, string>; onChanged: () => void }) {
  const c = useCopy().idol;
  const [rows, setRows] = useState<any[]>([]);
  async function load() {
    if (!props.idol) return;
    const m = await api.idolCall("memories", { idol_id: props.idol.id });
    setRows(asArray(m?.memories));
  }
  useEffect(() => { void load().catch((e) => toast.error(idolError(e, c))); }, [props.idol?.id]);
  if (!props.idol) return <p className="text-[13px] text-muted">{c.empty}</p>;
  if (!rows.length) return <p className="text-[13px] text-muted">{c.memoryEmpty}</p>;
  return (
    <ul className="space-y-2">
      {rows.map((m) => {
        const from = str(m.source_idol);
        const shared = from && from !== props.idol?.id;
        return (
          <li key={str(m.id)} className="rounded-xl border border-border px-3 py-2 text-[13px]">
            <p className={m.superseded_by ? "text-muted line-through" : ""}>{str(m.text)}</p>
            <div className="mt-1 flex items-center justify-between gap-2 text-[11px] text-muted">
              <span>{shared ? c.memoryFrom.replace("{name}", props.names[from] || from) : ""} {clock(str(m.at))}</span>
              <button type="button" onClick={() => void api.idolCall("delete_memory", { id: m.id }).then(() => { props.onChanged(); return load(); })}>{c.deleteMemory}</button>
            </div>
          </li>
        );
      })}
    </ul>
  );
}

function Gifts(props: { idol: Idol | null; onChanged: () => void; onSave: (i: Idol) => void }) {
  const c = useCopy().idol;
  const [gifts, setGifts] = useState<any>(null);
  const [note, setNote] = useState("");
  async function load() {
    setGifts(await api.idolCall("gifts"));
  }
  useEffect(() => { void load().catch((e) => toast.error(idolError(e, c))); }, []);
  async function act(op: string, extra?: Record<string, unknown>) {
    setNote("");
    try {
      const res = await api.idolCall(op, extra);
      if (op === "draw") setNote(res?.reset_card ? c.drewReset : c.drawn.replace("{name}", str(res?.gift?.name || "")));
      if (op === "give") {
        setNote(c.giftSent);
        props.onChanged();
      }
      if (op === "reset") setNote(c.reset);
      await load();
    } catch (e) {
      toast.error(idolError(e, c));
    }
  }
  const liked = new Set(props.idol?.liked_gifts || []);
  return (
    <div className="space-y-3 text-[13px]">
      <div className="flex flex-wrap items-center gap-2">
        <span>{c.draws}: {gifts?.draws ?? 0}</span>
        <Button size="sm" variant="lift" onClick={() => void act("draw")}>{c.draw}</Button>
        <Button size="sm" variant="lift" onClick={() => void act("reset")}>{c.reset} · {gifts?.reset_cards ?? 0}</Button>
      </div>
      {note ? <p className="text-[12px] text-muted">{note}</p> : null}
      <div className="grid grid-cols-2 gap-2">
        {asArray(gifts?.catalog).map((g: any) => {
          const held = num(asArray(gifts?.inventory).find((s: any) => s.gift_id === g.id)?.count, 0);
          const love = liked.has(g.id);
          return (
            <div key={g.id} className="rounded-xl border border-border p-3">
              <div className="font-medium">{g.name}</div>
              <div className="mt-1 text-[11px] text-muted">{(c.rarity as any)[g.rarity] || g.rarity} · {c.held} {held}</div>
              <div className="mt-2 flex gap-1">
                <Button size="sm" variant="lift" disabled={!held || !props.idol} onClick={() => void act("give", { idol_id: props.idol?.id, gift_id: g.id })}>{c.give}</Button>
                {props.idol ? (
                  <Button size="sm" variant={love ? "default" : "ghost"} onClick={() => {
                    const next = love ? props.idol!.liked_gifts.filter((id) => id !== g.id) : [...props.idol!.liked_gifts, g.id];
                    props.onSave({ ...props.idol!, liked_gifts: next });
                  }}>{love ? c.liked : c.likedMark}</Button>
                ) : null}
              </div>
            </div>
          );
        })}
      </div>
    </div>
  );
}

function LorePanel(props: { idol: Idol | null; lore: Lore[]; onChanged: () => void }) {
  const c = useCopy().idol;
  const [title, setTitle] = useState("");
  const [body, setBody] = useState("");
  const rows = props.lore.filter((d) => d.idol_id === props.idol?.id);
  if (!props.idol) return <p className="text-[13px] text-muted">{c.empty}</p>;
  return (
    <div className="space-y-3">
      {rows.map((d) => (
        <article key={d.id} className="rounded-xl border border-border p-3">
          <div className="text-[13px] font-medium">{d.title}</div>
          <p className="mt-1 whitespace-pre-wrap text-[12px] text-muted">{d.body}</p>
          <button type="button" className="mt-2 text-[12px] text-muted" onClick={() => void api.idolCall("delete_lore", { id: d.id }).then(props.onChanged)}>{c.delete}</button>
        </article>
      ))}
      <Input value={title} placeholder={c.loreTitle} onChange={(e) => setTitle(e.target.value)} />
      <textarea className="min-h-24 w-full rounded-md border border-border bg-background p-2 text-[13px]" value={body} placeholder={c.loreBody} onChange={(e) => setBody(e.target.value)} />
      <Button size="sm" disabled={!title.trim() && !body.trim()} onClick={() => void api.idolCall("save_lore", { idol_id: props.idol!.id, title, body }).then(() => { setTitle(""); setBody(""); props.onChanged(); })}>{c.addLore}</Button>
    </div>
  );
}

function Stats(props: { names: Record<string, string>; idol: Idol | null }) {
  const c = useCopy().idol;
  const [stats, setStats] = useState<any>(null);
  useEffect(() => { void api.idolCall("stats").then(setStats).catch((e) => toast.error(idolError(e, c))); }, []);
  if (!stats) return <p className="text-[13px] text-muted">{c.loading}</p>;
  const tasks = stats.tasks || {};
  const ach = asArray(stats.achievements).map((id) => achievementLabel(str(id), props.names, c.ach as unknown as Record<string, string>, c.achClose, c.achIntimate));
  return (
    <div className="space-y-4 text-[13px]">
      <section>
        <h3 className="mb-2 text-[12px] text-muted">{c.today}</h3>
        <ul className="space-y-1">
          <li>{tasks.talk ? "✓" : "·"} {c.taskTalk}</li>
          <li>{tasks.call ? "✓" : "·"} {c.taskCall}</li>
          <li>{tasks.gift ? "✓" : "·"} {c.taskGift}</li>
        </ul>
      </section>
      <section>
        <h3 className="mb-2 text-[12px] text-muted">{c.statsAll}</h3>
        <div className="grid grid-cols-2 gap-2">
          <Stat label={c.turns} value={stats.turns} />
          <Stat label={c.callTime} value={`${stats.call_sec || 0}s · ${stats.calls || 0}`} />
          <Stat label={c.giftsGiven} value={stats.gifts} />
          <Stat label={c.memories} value={stats.memories} />
          <Stat label={c.draws} value={stats.draws_left} />
          <Stat label={c.reset} value={stats.reset_cards} />
        </div>
      </section>
      <section>
        <h3 className="mb-2 text-[12px] text-muted">{c.achievements}</h3>
        {ach.length ? <ul className="space-y-1">{ach.map((a) => <li key={a}>{a}</li>)}</ul> : <p className="text-muted">—</p>}
      </section>
    </div>
  );
}

function Stat(props: { label: string; value: string | number }) {
  return (
    <div className="rounded-xl bg-lift px-3 py-2">
      <div className="text-[11px] text-muted">{props.label}</div>
      <div className="tabular-nums">{props.value ?? "—"}</div>
    </div>
  );
}

function Circles(props: {
  circles: Circle[];
  idols: Idol[];
  idol: Idol | null;
  onSave: (i: Idol) => void;
  onChanged: () => void;
}) {
  const c = useCopy().idol;
  const [name, setName] = useState("");
  const [renaming, setRenaming] = useState("");
  const [rename, setRename] = useState("");
  async function saveCircle(id: string, next: string) {
    const trimmed = next.trim();
    if (!trimmed) return;
    await api.idolCall("save_circle", { id, name: trimmed });
    setName("");
    setRenaming("");
    props.onChanged();
  }
  return (
    <div className="space-y-3 text-[13px]">
      <p className="leading-relaxed text-muted">{c.circleHint}</p>
      {props.circles.length === 0 ? <p className="text-muted">{c.circleEmpty}</p> : null}
      {props.circles.map((cir) => {
        const members = props.idols.filter((i) => i.circle_id === cir.id && !i.hidden);
        const here = props.idol?.circle_id === cir.id;
        return (
          <article key={cir.id} className="rounded-xl border border-border p-3">
            {renaming === cir.id ? (
              <form className="flex gap-2" onSubmit={(e) => { e.preventDefault(); void saveCircle(cir.id, rename); }}>
                <Input className="h-8" value={rename} onChange={(e) => setRename(e.target.value)} />
                <Button size="sm" type="submit">{c.save}</Button>
              </form>
            ) : <div className="font-medium">{cir.name}</div>}
            <p className="mt-1 text-[12px] text-muted">{c.members.replace("{n}", String(members.length))}{members.length ? ` · ${members.map((m) => m.name).join("、")}` : ""}</p>
            <div className="mt-2 flex flex-wrap gap-1">
              {props.idol ? (
                <Button size="sm" variant={here ? "default" : "lift"} onClick={() => props.onSave({ ...props.idol!, circle_id: here ? "" : cir.id })}>
                  {here ? c.circleLeave : c.circleJoin}
                </Button>
              ) : null}
              <Button size="sm" variant="ghost" onClick={() => { setRenaming(cir.id); setRename(cir.name); }}>{c.circleRename}</Button>
              <Button size="sm" variant="ghost" onClick={() => void api.idolCall("delete_circle", { id: cir.id }).then(props.onChanged)}>{c.circleDelete}</Button>
            </div>
          </article>
        );
      })}
      <form className="flex gap-2" onSubmit={(e) => { e.preventDefault(); void saveCircle("", name); }}>
        <Input className="h-8" value={name} placeholder={c.circleNamePh} onChange={(e) => setName(e.target.value)} />
        <Button size="sm" type="submit" disabled={!name.trim()}>{c.createCircle}</Button>
      </form>
    </div>
  );
}

function HistoryList(props: {
  history: { current: Conversation | null; past: Conversation[] };
  viewing: string;
  onView: (id: string) => void;
  onDeleteChat: (id: string) => void;
  onClose: () => void;
}) {
  const c = useCopy().idol;
  const rows = props.history.past;
  if (!rows.length) return <p className="text-[13px] text-muted">{c.pastEmpty}</p>;
  return (
    <ul className="space-y-2">
      {rows.map((p) => (
        <li key={p.id} className="rounded-xl border border-border p-3">
          <button type="button" className="block w-full text-left" onClick={() => { props.onView(p.id); props.onClose(); }}>
            <div className="text-[13px] font-medium">{p.title || whenLabel(p.created_at)}</div>
            <div className="text-[11px] text-muted">{whenLabel(p.created_at)} · {p.records.length}</div>
          </button>
          <button type="button" className="mt-2 text-[12px] text-muted" onClick={() => props.onDeleteChat(p.id)}>{c.deleteChat}</button>
          {props.viewing === p.id ? <div className="mt-1 text-[11px] text-muted">{c.readOnly}</div> : null}
        </li>
      ))}
    </ul>
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

function Lang(props: { label: string; value: string; onChange: (v: string) => void }) {
  const options = ["zh", "en", "ja"];
  const extra = props.value && !options.includes(props.value) ? [props.value] : [];
  return (
    <label className="block text-[12px] text-muted">
      {props.label}
      <select className="mt-1 h-8 w-full rounded-md border border-border bg-background px-2 text-[13px]" value={props.value} onChange={(e) => props.onChange(e.target.value)}>
        {[...options, ...extra].map((o) => <option key={o} value={o}>{o}</option>)}
      </select>
    </label>
  );
}

export function emptyDraft() {
  return blankIdol();
}
