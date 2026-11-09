/**
 * 拥有：Buddy 工作区：顶部的说明和「刷新」「偏好」，没选中时的分组总览（和侧栏同样四组，用细线分隔），选中时的详情，
 * 以及栏目里全部对话框和侧板：一键配置、装到系统的确认（显示完整命令）、卸载确认、登录命令、日志、偏好、添加自定义 agent。
 * 不拥有：侧栏（BuddyRailNav.tsx）和动作（buddy-actions.ts）。打开时立刻取一次列表。
 */

import { Bot, Loader2, Plus, RefreshCw, Settings2 } from "lucide-react";
import { useEffect, useMemo, useState } from "react";
import { toast } from "sonner";
import { Button } from "../../components/ui/button";
import { EmptyState } from "../../components/ui/empty-state";
import { Input, Textarea } from "../../components/ui/input";
import { buddy, type BuddyView } from "../../lib/client";
import { writeClipboard } from "../../lib/clipboard";
import { useCopy } from "../../lib/i18n";
import { useUI } from "../../lib/store";
import { ConfirmDialog } from "../ConfirmDialog";
import { FormDialog } from "../FormDialog";
import { BuddyConfigDialog } from "./BuddyConfigDialog";
import { BuddyDetail } from "./BuddyDetail";
import { BuddyLogSheet } from "./BuddyLogSheet";
import { BuddyPrefsSheet } from "./BuddyPrefsSheet";
import { BuddyRow } from "./BuddyRow";
import { buddyDo, buddySystemInstall, buddyUninstall } from "./buddy-actions";
import { buddyPrimary, buddySections, fill, hasExternal, parseEnv, splitLines, type BuddyCopy } from "./buddy-model";
import { errText, useBuddy } from "./buddy-store";

/** RECOMMENDED 是一个外部 agent 都没装时推荐的三个。 */
const RECOMMENDED = ["claude", "codex", "gemini"];

const AREA =
  "min-h-[4.5rem] rounded-lg border border-border bg-background px-2.5 py-2 font-mono text-ui-footnote leading-5 focus-visible:border-accent/45 focus-visible:ring-2 focus-visible:ring-accent/20";

/** Skeleton 是第一次取数时的四行骨架。 */
function Skeleton({ c }: { c: BuddyCopy }) {
  return (
    <div role="status" aria-label={c.loading} className="border-t border-border/60" data-testid="buddy-skeleton">
      {[0, 1, 2, 3].map((i) => (
        <div key={i} className="flex min-h-14 items-center gap-3 border-b border-border/60 px-1 py-2">
          <div className="size-6 animate-pulse rounded-full bg-lift/80" />
          <div className="flex-1 space-y-1.5">
            <div className="h-3 w-40 animate-pulse rounded-sm bg-lift/80" />
            <div className="h-2.5 w-24 animate-pulse rounded-sm bg-lift/60" />
          </div>
        </div>
      ))}
    </div>
  );
}

/** Group 是总览里的一组：组名，下面是用细线分隔的行。空组不画。 */
function Group({ label, list, onOpen }: { label: string; list: BuddyView[]; onOpen: (id: string) => void }) {
  if (!list.length) return null;
  return (
    <section className="mt-6 first:mt-0" aria-label={label}>
      <h3 className="mb-1 text-ui-caption text-muted">{label}</h3>
      <div className="border-t border-border/60">
        {list.map((v) => (
          <BuddyRow key={v.id} v={v} onOpen={onOpen} />
        ))}
      </div>
    </section>
  );
}

/** Recommend 是空状态里的推荐安装按钮：没在装时直接装，在装时转圈。 */
function Recommend({ v, c }: { v: BuddyView; c: BuddyCopy }) {
  const action = buddyPrimary(v);
  const installing = v.state === "installing";
  return (
    <Button size="sm" variant="outline" disabled={installing || (action !== "install" && action !== "installSystem")} onClick={() => void buddyDo(v, action)}>
      {installing ? <Loader2 className="animate-spin" aria-hidden /> : null}
      {fill(c.installNamed, { name: v.name })}
    </Button>
  );
}

/** Overview 是没选中任何一项时的总览。一个外部 agent 都没有时先给空状态和三个推荐，推荐里还没装的不再在下面重复。 */
function Overview({ c }: { c: BuddyCopy }) {
  const list = useBuddy((s) => s.list);
  const ready = useBuddy((s) => s.loaded && !s.detecting);
  const error = useBuddy((s) => s.error);
  const openDialog = useBuddy((s) => s.openDialog);
  const setFocus = useUI((s) => s.setBuddyFocus);
  const [moreOpen, setMoreOpen] = useState(false);
  const empty = ready && !hasExternal(list);
  const sections = useMemo(() => {
    const s = buddySections(list);
    if (!empty) return s;
    const hide = (v: BuddyView) => RECOMMENDED.includes(v.id) && v.state === "not_installed";
    return { online: [], needs: s.needs, available: s.available.filter((v) => !hide(v)), more: s.more.filter((v) => !hide(v)) };
  }, [list, empty]);
  if (!ready) return error && !list.length ? <p role="alert" className="text-ui-body text-danger">{fill(c.loadFailed, { error })}</p> : <Skeleton c={c} />;
  const recommended = RECOMMENDED.map((id) => list.find((v) => v.id === id)).filter((v): v is BuddyView => !!v);
  return (
    <div data-testid="buddy-overview">
      {empty ? (
        <EmptyState
          className="py-10"
          icon={<Bot className="size-8" aria-hidden />}
          title={c.emptyTitle}
          body={c.emptyBody}
          action={
            recommended.length ? (
              <div className="flex flex-wrap justify-center gap-2">
                {recommended.map((v) => (
                  <Recommend key={v.id} v={v} c={c} />
                ))}
              </div>
            ) : undefined
          }
        />
      ) : null}
      <Group label={c.sectionOnline} list={sections.online} onOpen={setFocus} />
      <Group label={c.sectionNeeds} list={sections.needs} onOpen={setFocus} />
      <Group label={c.sectionAvailable} list={sections.available} onOpen={setFocus} />
      {sections.more.length && moreOpen ? <Group label={c.sectionMore} list={sections.more} onOpen={setFocus} /> : null}
      <div className="mt-4 flex flex-wrap items-center gap-2">
        {sections.more.length ? (
          <Button size="sm" variant="ghost" aria-expanded={moreOpen} onClick={() => setMoreOpen(!moreOpen)}>
            {moreOpen ? c.fewer : fill(c.moreCount, { n: sections.more.length })}
          </Button>
        ) : null}
        <Button size="sm" variant="ghost" onClick={() => openDialog({ kind: "custom" })}>
          <Plus aria-hidden />
          {c.addCustom}
        </Button>
      </div>
    </div>
  );
}

/** CustomForm 是添加自定义 agent 的表单：名字、命令、每行一个的参数和环境变量。加好后选中它，并等探测结果。 */
function CustomForm({ c, onClose }: { c: BuddyCopy; onClose: () => void }) {
  const setFocus = useUI((s) => s.setBuddyFocus);
  const [name, setName] = useState("");
  const [command, setCommand] = useState("");
  const [args, setArgs] = useState("");
  const [env, setEnv] = useState("");
  const [note, setNote] = useState("");
  const [busy, setBusy] = useState(false);
  return (
    <form
      className="flex flex-col gap-3"
      onSubmit={(e) => {
        e.preventDefault();
        const parsed = parseEnv(env);
        if (parsed.bad) {
          setNote(fill(c.detail.envBad, { line: parsed.bad }));
          return;
        }
        setBusy(true);
        setNote("");
        void buddy
          .add({ name: name.trim(), command: command.trim(), args: splitLines(args), env: parsed.env })
          .then((v) => {
            const s = useBuddy.getState();
            s.put(v);
            s.watch(v);
            setFocus(v.id);
            toast(fill(c.toast.added, { name: v.name }));
            onClose();
          })
          .catch((err) => setNote(errText(err)))
          .finally(() => setBusy(false));
      }}
    >
      <label className="block space-y-1">
        <span className="text-ui-footnote text-muted">{c.custom.name}</span>
        <Input value={name} onChange={(e) => setName(e.target.value)} required maxLength={60} />
      </label>
      <label className="block space-y-1">
        <span className="text-ui-footnote text-muted">{c.custom.command}</span>
        <Input value={command} onChange={(e) => setCommand(e.target.value)} required placeholder={c.custom.commandHint} className="font-mono" />
      </label>
      <label className="block space-y-1">
        <span className="text-ui-footnote text-muted">{c.custom.args}</span>
        <Textarea className={AREA} value={args} placeholder={c.custom.argsHint} onChange={(e) => setArgs(e.target.value)} spellCheck={false} />
      </label>
      <div className="space-y-1">
        <label className="block space-y-1">
          <span className="text-ui-footnote text-muted">{c.custom.env}</span>
          <Textarea className={AREA} value={env} placeholder="NAME=value" aria-describedby="buddy-custom-env-hint" onChange={(e) => setEnv(e.target.value)} spellCheck={false} />
        </label>
        <p id="buddy-custom-env-hint" className="text-ui-caption text-muted">{c.custom.envHint}</p>
      </div>
      {note ? <p role="alert" className="text-ui-footnote text-danger">{note}</p> : null}
      <div className="dialog-actions">
        <Button type="button" size="sm" variant="lift" onClick={onClose}>
          {c.action.cancel}
        </Button>
        <Button type="submit" size="sm" disabled={busy || !name.trim() || !command.trim()} aria-busy={busy || undefined}>
          {c.custom.add}
        </Button>
      </div>
    </form>
  );
}

/** CommandBlock 是一段要人看清楚的命令，右边可以复制。 */
function CommandBlock({ command, c }: { command: string; c: BuddyCopy }) {
  return (
    <div className="mt-3 flex items-start gap-2 rounded-lg border border-border bg-background p-2.5">
      <code className="min-w-0 flex-1 whitespace-pre-wrap break-all font-mono text-ui-footnote text-foreground" data-testid="buddy-command">
        {command}
      </code>
      <Button
        size="xs"
        variant="ghost"
        onClick={() => {
          void writeClipboard(command).then((ok) => ok && toast(c.detail.copied));
        }}
      >
        {c.confirm.copyCommand}
      </Button>
    </div>
  );
}

/** Dialogs 挂着 Buddy 的全部对话框和侧板，开哪个由仓库里的 dialog 决定。 */
function Dialogs({ c }: { c: BuddyCopy }) {
  const dialog = useBuddy((s) => s.dialog);
  const list = useBuddy((s) => s.list);
  const openDialog = useBuddy((s) => s.openDialog);
  const focus = useUI((s) => s.buddyFocus);
  const setFocus = useUI((s) => s.setBuddyFocus);
  const close = () => openDialog(null);
  const target = dialog && "id" in dialog ? list.find((v) => v.id === dialog.id) : undefined;
  const pick = (kind: string) => (dialog?.kind === kind ? target : undefined);
  const system = pick("system");
  const removing = pick("uninstall");
  const loginCopy = dialog?.kind === "loginCopy" ? dialog.command : "";
  return (
    <>
      <BuddyConfigDialog v={pick("setup")} onClose={close} />
      <ConfirmDialog
        open={!!system}
        title={system ? fill(c.confirm.systemTitle, { name: system.name }) : ""}
        body={
          system ? (
            <>
              <p>{fill(c.confirm.systemBody, { name: system.name })}</p>
              <CommandBlock command={system.systemInstall?.command || ""} c={c} />
            </>
          ) : null
        }
        confirmLabel={c.confirm.systemRun}
        onCancel={close}
        onConfirm={async () => {
          if (!system) return;
          close();
          await buddySystemInstall(system);
        }}
      />
      <ConfirmDialog
        open={!!removing}
        danger
        title={removing ? fill(removing.custom ? c.confirm.removeTitle : c.confirm.uninstallTitle, { name: removing.name }) : ""}
        body={
          removing
            ? fill(removing.custom ? c.confirm.removeBody : removing.install.kind === "system" ? c.confirm.uninstallSystemBody : c.confirm.uninstallBody, { name: removing.name })
            : ""
        }
        confirmLabel={removing?.custom ? c.action.remove : c.action.uninstall}
        onCancel={close}
        onConfirm={async () => {
          if (!removing) return;
          await buddyUninstall(removing);
          close();
          if (removing.custom && focus === removing.id) setFocus("");
        }}
      />
      <ConfirmDialog
        open={!!loginCopy}
        title={c.detail.login}
        body={
          <>
            <p>{c.detail.loginCopy}</p>
            <CommandBlock command={loginCopy} c={c} />
          </>
        }
        confirmLabel={c.confirm.copyCommand}
        onCancel={close}
        onConfirm={async () => {
          if (await writeClipboard(loginCopy)) toast(c.detail.copied);
          close();
        }}
      />
      <BuddyLogSheet v={pick("log")} onClose={close} />
      <BuddyPrefsSheet open={dialog?.kind === "prefs"} onClose={close} />
      <FormDialog open={dialog?.kind === "custom"} title={c.custom.title} description={c.custom.body} onClose={close}>
        {dialog?.kind === "custom" ? <CustomForm c={c} onClose={close} /> : null}
      </FormDialog>
    </>
  );
}

/** BuddyWorkspace 是 Buddy 表面的右侧。选中的项找不到时（比如刚移除）回到总览。 */
export function BuddyWorkspace() {
  const c = useCopy().buddy;
  const list = useBuddy((s) => s.list);
  const openDialog = useBuddy((s) => s.openDialog);
  const focus = useUI((s) => s.buddyFocus);
  const [refreshing, setRefreshing] = useState(false);
  const focused = focus ? list.find((v) => v.id === focus) : undefined;
  useEffect(() => {
    void useBuddy.getState().load();
  }, []);
  return (
    <div className="flex h-full min-h-0 flex-col" data-testid="buddy-workspace">
      <div className="min-h-0 flex-1 overflow-auto">
        {focused ? (
          <BuddyDetail key={focused.id} v={focused} />
        ) : (
          <div className="mx-auto w-full max-w-4xl px-6 py-6">
            <div className="mb-5 flex flex-wrap items-start gap-3">
              <p className="min-w-[16rem] flex-1 text-ui-body text-muted">{c.intro}</p>
              <div className="flex shrink-0 items-center gap-1">
                <Button
                  size="sm"
                  variant="outline"
                  disabled={refreshing}
                  onClick={() => {
                    setRefreshing(true);
                    void useBuddy
                      .getState()
                      .load("force")
                      .finally(() => setRefreshing(false));
                  }}
                >
                  <RefreshCw className={refreshing ? "animate-spin" : undefined} aria-hidden />
                  {c.refresh}
                </Button>
                <Button size="sm" variant="ghost" onClick={() => openDialog({ kind: "prefs" })}>
                  <Settings2 aria-hidden />
                  {c.prefs}
                </Button>
              </div>
            </div>
            <Overview c={c} />
          </div>
        )}
      </div>
      <Dialogs c={c} />
    </div>
  );
}
