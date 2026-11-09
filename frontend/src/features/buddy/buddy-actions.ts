/**
 * 拥有：Buddy 栏目里每个按钮和菜单项背后的动作：调哪个 buddy.* 接口、结果怎么写回仓库、要不要等结果再提示、该开哪个对话框。
 * 不拥有：按钮长什么样，也不判断该出现哪些按钮（buddy-model.ts）。行、详情和对话框都调这里。
 */

import { toast } from "sonner";
import { buddy, openPath, openURL, type BuddyConfigPatch, type BuddyView } from "../../lib/client";
import { copy } from "../../lib/copy";
import type { BuddyAction, BuddyMenuItem } from "./buddy-model";
import { fill } from "./buddy-model";
import { useBuddy } from "./buddy-store";

/** install 用 mode 安装或更新 v，并等它装好后提示。 */
async function install(v: BuddyView, mode: "managed" | "system") {
  const s = useBuddy.getState();
  const next = await s.run(v.id, "install", () => buddy.install(v.id, mode));
  if (!next) return;
  s.put(next);
  s.watch(next);
}

/** probe 做一次全探，按结果提示：在线说就绪，否则说原因。 */
async function probe(v: BuddyView) {
  const s = useBuddy.getState();
  const next = await s.run(v.id, "probe", () => buddy.probe(v.id));
  if (!next) return;
  s.put(next);
  const c = copy.buddy;
  if (next.state === "online") toast.success(fill(c.toast.ready, { name: next.name }));
  else toast.error(fill(c.toast.failed, { name: next.name, error: next.detail || c.state[next.state] }));
}

/** buddyLogin 用 method（空时由后端挑第一个）登录。打不开终端时弹出命令给人复制；否则等它变成在线后提示。 */
export async function buddyLogin(v: BuddyView, method = "") {
  const s = useBuddy.getState();
  const res = await s.run(v.id, "login", () => buddy.login(v.id, method));
  if (!res) return;
  s.put(res.buddy);
  if (res.copy) {
    s.openDialog({ kind: "loginCopy", id: v.id, command: res.copy });
    return;
  }
  toast(fill(copy.buddy.detail.loginStarted, { name: v.name }));
  s.watch(res.buddy);
}

/** buddyConfigure 保存一次配置。watch 为真时等全探结果再提示。返回保存后的视图，失败时返回 undefined。 */
export async function buddyConfigure(v: BuddyView, patch: BuddyConfigPatch, watch: boolean): Promise<BuddyView | undefined> {
  const s = useBuddy.getState();
  const next = await s.run(v.id, "configure", () => buddy.configure(v.id, patch));
  if (!next) return undefined;
  s.put(next);
  if (watch) s.watch(next);
  return next;
}

/** buddyUninstall 卸载或移除 v。自定义 agent 移除后从列表里拿掉。 */
export async function buddyUninstall(v: BuddyView) {
  const s = useBuddy.getState();
  const next = await s.run(v.id, "uninstall", () => buddy.uninstall(v.id));
  if (!next) return;
  const c = copy.buddy;
  if (v.custom) {
    s.drop(v.id);
    toast(fill(c.toast.removed, { name: v.name }));
    return;
  }
  s.put({ ...v, ...next, name: next.name || v.name });
  toast(fill(c.toast.uninstalled, { name: v.name }));
}

/** buddySystemInstall 在人确认过完整命令之后装到系统。 */
export async function buddySystemInstall(v: BuddyView) {
  await install(v, "system");
}

/** buddyDo 执行主按钮或更多菜单里的一项。要确认或要填东西的动作只打开对应的对话框。 */
export async function buddyDo(v: BuddyView, action: BuddyAction | BuddyMenuItem) {
  const s = useBuddy.getState();
  switch (action) {
    case "install":
    case "update":
      return install(v, "managed");
    case "installSystem":
      return s.openDialog({ kind: "system", id: v.id });
    case "cancel":
      s.unwatch(v.id);
      await s.run(v.id, "cancel", () => buddy.cancel(v.id));
      return s.load();
    case "login":
      return buddyLogin(v);
    case "configure":
      return s.openDialog({ kind: "setup", id: v.id });
    case "retry":
    case "test":
    case "recheck":
      return probe(v);
    case "useSystem":
      await buddyConfigure(v, { mode: "system" }, false);
      return;
    case "useManaged": {
      const next = await buddyConfigure(v, { mode: "managed" }, false);
      if (next && next.install.kind !== "managed") await install(next, "managed");
      return;
    }
    case "openDir":
      if (v.install.path) await openPath(v.install.path).catch((e) => toast.error(String(e)));
      return;
    case "logs":
      return s.openDialog({ kind: "log", id: v.id });
    case "docs":
      if (v.docs) await openURL(v.docs).catch((e) => toast.error(String(e)));
      return;
    case "uninstall":
    case "remove":
      return s.openDialog({ kind: "uninstall", id: v.id });
    case "none":
      return;
  }
}
