/**
 * 拥有：Buddy 偏好侧板：npm 源、Python 源、默认 agent、「没 @ 时让路由自己挑」开关、单轮时长上限，以及磁盘占用和清理缓存。
 * 不拥有：取值校验（后端 buddy.prefs 会拒绝不合格的地址和时长）。打开时才去量磁盘，因为要把目录走一遍。
 */

import { X } from "lucide-react";
import { useEffect, useState } from "react";
import { toast } from "sonner";
import { Button } from "../../components/ui/button";
import { Input } from "../../components/ui/input";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "../../components/ui/select";
import { Sheet, SheetClose, SheetContent, SheetDescription, SheetTitle } from "../../components/ui/sheet";
import { Switch } from "../../components/ui/switch";
import { buddy, type BuddyDisk, type BuddyPrefs } from "../../lib/client";
import { useCopy } from "../../lib/i18n";
import { canDispatch, fill, formatSize, installed, type BuddyCopy } from "./buddy-model";
import { errText, useBuddy } from "./buddy-store";

const NPM_PRESETS = ["auto", "npmjs", "npmmirror"];
const FIELD = "space-y-1.5";
const FIELD_LABEL = "block text-ui-footnote text-muted";

/** Disk 是磁盘占用一段：总量、运行时和缓存，以及清理缓存按钮。 */
function Disk({ c }: { c: BuddyCopy }) {
  const [disk, setDisk] = useState<BuddyDisk | null>(null);
  const [busy, setBusy] = useState(false);
  useEffect(() => {
    let alive = true;
    void buddy.disk().then((d) => alive && setDisk(d)).catch(() => {});
    return () => {
      alive = false;
    };
  }, []);
  return (
    <div className="space-y-2 border-t border-border/60 pt-4">
      <p className="text-ui-body font-medium text-foreground">{c.pref.disk}</p>
      {disk ? (
        <>
          <p className="text-ui-footnote tabular-nums text-foreground">{fill(c.pref.diskTotal, { size: formatSize(disk.total) })}</p>
          <dl className="grid grid-cols-[1fr_auto] gap-x-4 gap-y-1 text-ui-footnote text-muted">
            <dt>{c.pref.diskRuntime}</dt>
            <dd className="tabular-nums">{formatSize(disk.runtime)}</dd>
            <dt>{c.pref.diskCache}</dt>
            <dd className="tabular-nums">{formatSize(disk.cache)}</dd>
          </dl>
        </>
      ) : (
        <div className="h-12 animate-pulse rounded-md bg-lift/70" />
      )}
      <Button
        size="sm"
        variant="outline"
        disabled={busy || !disk?.cache}
        onClick={() => {
          setBusy(true);
          void buddy
            .clearCache()
            .then((res) => {
              toast(fill(c.toast.cleared, { size: formatSize(res.freed) }));
              return buddy.disk().then(setDisk);
            })
            .catch((e) => toast.error(errText(e)))
            .finally(() => setBusy(false));
        }}
      >
        {c.pref.clearCache}
      </Button>
    </div>
  );
}

/** PrefsForm 是偏好表单。初值取自打开时的偏好，保存后写回仓库。 */
function PrefsForm({ prefs, c }: { prefs: BuddyPrefs; c: BuddyCopy }) {
  const list = useBuddy((s) => s.list);
  const setPrefs = useBuddy((s) => s.setPrefs);
  const custom = !NPM_PRESETS.includes(prefs.npmRegistry);
  const [npm, setNpm] = useState(custom ? "custom" : prefs.npmRegistry || "auto");
  const [npmURL, setNpmURL] = useState(custom ? prefs.npmRegistry : "");
  const [py, setPy] = useState(prefs.pyIndex);
  const [agent, setAgent] = useState(prefs.defaultAgent || "yoyo");
  const [autoPick, setAutoPick] = useState(prefs.autoPick);
  const [minutes, setMinutes] = useState(String(prefs.turnMinutes));
  const [note, setNote] = useState("");
  const [saving, setSaving] = useState(false);
  const agents = list.filter((v) => v.builtin || (installed(v) && v.config.enabled));
  const pick = agents.some((v) => v.id === agent) ? agent : "yoyo";
  return (
    <form
      className="space-y-4"
      onSubmit={(e) => {
        e.preventDefault();
        const patch: Partial<BuddyPrefs> = {};
        const registry = npm === "custom" ? npmURL.trim() : npm;
        if (registry !== prefs.npmRegistry) patch.npmRegistry = registry;
        if (py.trim() !== prefs.pyIndex) patch.pyIndex = py.trim();
        if (pick !== prefs.defaultAgent) patch.defaultAgent = pick;
        if (autoPick !== prefs.autoPick) patch.autoPick = autoPick;
        const n = Number.parseInt(minutes, 10);
        if (Number.isFinite(n) && n !== prefs.turnMinutes) patch.turnMinutes = n;
        setSaving(true);
        setNote("");
        void buddy
          .prefs(patch)
          .then((next) => {
            setPrefs(next);
            setNote(c.pref.saved);
          })
          .catch((err) => setNote(errText(err)))
          .finally(() => setSaving(false));
      }}
    >
      <div className={FIELD}>
        <label htmlFor="buddy-npm" className={FIELD_LABEL}>{c.pref.npm}</label>
        <Select value={npm} onValueChange={setNpm}>
          <SelectTrigger id="buddy-npm" className="h-8 w-full" aria-label={c.pref.npm}>
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="auto">{c.pref.npmAuto}</SelectItem>
            <SelectItem value="npmjs">{c.pref.npmOfficial}</SelectItem>
            <SelectItem value="npmmirror">{c.pref.npmMirror}</SelectItem>
            <SelectItem value="custom">{c.pref.npmCustom}</SelectItem>
          </SelectContent>
        </Select>
        {npm === "custom" ? (
          <Input aria-label={c.pref.npmCustom} value={npmURL} placeholder={c.pref.npmCustomHint} onChange={(e) => setNpmURL(e.target.value)} inputMode="url" />
        ) : null}
      </div>
      <div className={FIELD}>
        <label htmlFor="buddy-py" className={FIELD_LABEL}>{c.pref.py}</label>
        <Input id="buddy-py" value={py} placeholder={c.pref.pyHint} onChange={(e) => setPy(e.target.value)} inputMode="url" />
      </div>
      <div className={FIELD}>
        <label htmlFor="buddy-default" className={FIELD_LABEL}>{c.pref.defaultAgent}</label>
        <Select value={pick} onValueChange={setAgent}>
          <SelectTrigger id="buddy-default" className="h-8 w-full" aria-label={c.pref.defaultAgent}>
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            {agents.map((v) => (
              <SelectItem key={v.id} value={v.id}>
                {v.name}
                {canDispatch(v) ? "" : ` · ${c.state[v.state]}`}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
        <p className="text-ui-caption text-muted">{c.pref.defaultAgentHint}</p>
      </div>
      <label className="flex items-center justify-between gap-3 text-ui-body text-foreground">
        <span>{c.pref.autoPick}</span>
        <Switch checked={autoPick} onCheckedChange={setAutoPick} aria-label={c.pref.autoPick} />
      </label>
      <div className={FIELD}>
        <label htmlFor="buddy-minutes" className={FIELD_LABEL}>{c.pref.turnMinutes}</label>
        <Input id="buddy-minutes" type="number" min={5} max={600} step={5} className="w-28 tabular-nums" value={minutes} onChange={(e) => setMinutes(e.target.value)} />
      </div>
      <div className="flex items-center gap-3">
        <Button type="submit" size="sm" disabled={saving} aria-busy={saving || undefined}>
          {c.pref.save}
        </Button>
        {note ? (
          <span role="status" className={note === c.pref.saved ? "text-ui-footnote text-status-done" : "text-ui-footnote text-danger"}>
            {note}
          </span>
        ) : null}
      </div>
    </form>
  );
}

/** BuddyPrefsSheet 是从右边滑出的偏好侧板。 */
export function BuddyPrefsSheet({ open, onClose }: { open: boolean; onClose: () => void }) {
  const c = useCopy().buddy;
  const prefs = useBuddy((s) => s.prefs);
  return (
    <Sheet open={open} onOpenChange={(next) => !next && onClose()}>
      <SheetContent side="right" className="flex flex-col" data-testid="buddy-prefs">
        <div className="flex h-12 shrink-0 items-center justify-between border-b border-border px-4">
          <SheetTitle className="text-ui-body font-medium">{c.pref.title}</SheetTitle>
          <SheetClose className="grid size-8 place-items-center rounded-lg text-muted hover:bg-lift hover:text-foreground" aria-label={c.action.cancel}>
            <X className="size-4" />
          </SheetClose>
        </div>
        <div className="min-h-0 flex-1 space-y-5 overflow-auto px-4 py-4">
          <SheetDescription className="text-ui-footnote text-muted">{c.pref.body}</SheetDescription>
          {prefs ? <PrefsForm prefs={prefs} c={c} /> : null}
          {open ? <Disk c={c} /> : null}
        </div>
      </SheetContent>
    </Sheet>
  );
}
