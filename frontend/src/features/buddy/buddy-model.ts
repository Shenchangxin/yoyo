/**
 * 拥有：Buddy 页里和画法无关的判断：状态的色调和说法、主按钮、更多菜单、五个分组（含「在这台电脑上发现」）、能不能派任务、安装进度怎么讲，以及表单文本的解析。
 * 不拥有：取数和轮询（buddy-store.ts），也不拥有任何组件。文案由调用方传进来，这里不读当前语言。
 */

import type { BuddySecret, BuddyView } from "../../lib/client.ts";
import type { Copy } from "../../lib/copy.ts";

/** BuddyCopy 是文案表里的 buddy 一节。 */
export type BuddyCopy = Copy["buddy"];

/** BuddyTone 是 StatusDot 的色调。still 为假时 live 带呼吸外圈。 */
export type BuddyTone = { tone: "needs" | "live" | "done" | "failed" | "idle"; still: boolean };

/** BuddyAction 是一行的主操作。none 表示不放按钮。 */
export type BuddyAction = "install" | "installSystem" | "cancel" | "login" | "configure" | "retry" | "test" | "none";

/** BuddyMenuItem 是更多菜单里的一项。 */
export type BuddyMenuItem = "installSystem" | "useManaged" | "useSystem" | "update" | "recheck" | "openDir" | "logs" | "docs" | "uninstall" | "remove";

/**
 * BuddySections 是栏目的五个分组：需要处理、在这台电脑上发现、在线、可安装、更多。
 * found 是本机已经有它的东西、但还不能直接派的：只有命令行缺适配器，或者只装了桌面应用。
 */
export type BuddySections = { online: BuddyView[]; needs: BuddyView[]; found: BuddyView[]; available: BuddyView[]; more: BuddyView[] };

/** BuddyBar 是安装进度细线的画法。indeterminate 为真时不画宽度。 */
export type BuddyBar = { value: number; indeterminate: boolean };

const NEEDS_YOU = new Set(["needs_auth", "needs_config", "error", "offline"]);

/** fill 把模板里的 {名字} 换成 vars 里的值，没给的占位原样留着。 */
export function fill(template: string, vars: Record<string, string | number>): string {
  return template.replace(/\{(\w+)\}/g, (all, key: string) => (key in vars ? String(vars[key]) : all));
}

/** installed 表示这一项已经装好：托管、系统或自定义。内置的 yoyo 不算。 */
export function installed(v: BuddyView): boolean {
  return !v.builtin && v.install.kind !== "none";
}

/**
 * foundOnHost 表示这台电脑上已经有它的东西，但还不能直接派任务：只有命令行、缺 yoyo 对接用的适配器，
 * 或者只装了桌面应用。已经能连的（装好的、厂商安装器放在别处的命令）不算，它们已经在别的分组里。
 */
export function foundOnHost(v: BuddyView): boolean {
  if (v.builtin || installed(v) || v.state !== "not_installed") return false;
  return !!v.cliOnly || v.host?.kind === "cli" || v.host?.kind === "app";
}

/** hostNote 说清在这台电脑上找到的是什么、还差什么：只装了桌面应用；桌面应用里带着命令行；或者 PATH 上有命令行。 */
export function hostNote(v: BuddyView, c: BuddyCopy): string {
  if (v.host?.kind === "app") return fill(c.appOnly, { app: v.host.app || v.name, name: v.name });
  if (v.host?.kind === "cli" && v.host.app) return fill(c.cliOnlyApp, { app: v.host.app });
  return c.cliOnly;
}

/** busy 表示正在安装或探测。这时界面每秒轮询一次。 */
export function busy(v: BuddyView): boolean {
  return v.state === "installing" || v.state === "probing";
}

/** canDispatch 表示 yoyo 现在能把任务派给它：内置的 yoyo 永远能；其余要启用并且在线。和 Go 的 Manager.Ready 一致。 */
export function canDispatch(v: BuddyView): boolean {
  return !!v.builtin || (v.config.enabled && v.state === "online");
}

/** routable 表示路由现在能把工单交给它，和 Go 的 blockedReason 一致：yoyo 永远能；其余要装好、启用，并且在线、检查中或刚断开。 */
export function routable(v: BuddyView): boolean {
  if (v.builtin) return true;
  if (!installed(v) || !v.config.enabled) return false;
  return v.state === "online" || v.state === "probing" || v.state === "recently_lost";
}

/** matchesAgent 在 q 为空，或编号、名字、厂商、别名里有一个包含 q 时为真，不分大小写。 */
export function matchesAgent(v: BuddyView, q: string): boolean {
  const needle = q.trim().toLowerCase();
  if (!needle) return true;
  return [v.id, v.name, v.vendor, ...(v.aliases ?? [])].some((s) => !!s && s.toLowerCase().includes(needle));
}

/** mentionAgents 是输入框 @ 菜单里的 agent。一个外部 agent 都没装时没有。yoyo 在最前，能派的排在不能派的前面，各自按 sortBuddies。 */
export function mentionAgents(list: BuddyView[], q: string): BuddyView[] {
  if (!hasExternal(list)) return [];
  const rows = sortBuddies(list).filter((v) => (v.builtin || installed(v)) && matchesAgent(v, q));
  return [...rows.filter(routable), ...rows.filter((v) => !routable(v))];
}

/** agentStateText 是 @ 菜单和任务页上的短状态：在做几个、停用，或者状态本身，不带原因。 */
export function agentStateText(v: BuddyView, c: BuddyCopy): string {
  if ((v.working ?? 0) > 0) return fill(c.state.working, { n: v.working ?? 0 });
  if (v.builtin) return c.state.online;
  if (installed(v) && !v.config.enabled) return c.disabled;
  return c.state[v.state] ?? v.state;
}

/** BlockedCopy 是点了派不了的 agent 时的几句提示，模板里的 {name} 换成名字。 */
export type BlockedCopy = { agentSignIn: string; agentSetup: string; agentTurnOn: string; agentUnavailable: string };

/** blockedHint 说清这一项要先怎样才能派：启用、登录、配置，否则只说现在接不了。 */
export function blockedHint(v: BuddyView, c: BlockedCopy): string {
  if (installed(v) && !v.config.enabled) return fill(c.agentTurnOn, { name: v.name });
  if (v.state === "needs_auth") return fill(c.agentSignIn, { name: v.name });
  if (v.state === "needs_config") return fill(c.agentSetup, { name: v.name });
  return fill(c.agentUnavailable, { name: v.name });
}

/** needsYou 表示这一项启用着，却因为要登录、要配置、出错或离线而接不了任务。dock 角标数的就是它。 */
export function needsYou(v: BuddyView): boolean {
  return installed(v) && v.config.enabled && (v.working ?? 0) === 0 && NEEDS_YOU.has(v.state);
}

/** attentionCount 是 dock 角标的数字。 */
export function attentionCount(list: BuddyView[]): number {
  return list.filter(needsYou).length;
}

/** anyWorking 表示有 agent 正在做任务，dock 上亮 live 标记。 */
export function anyWorking(list: BuddyView[]): boolean {
  return list.some((v) => (v.working ?? 0) > 0);
}

/** onlineCount 是现在能接任务的个数，yoyo 也算。 */
export function onlineCount(list: BuddyView[]): number {
  return list.filter(canDispatch).length;
}

/** buddyTone 把状态映射到 StatusDot：在做任务用 live 动画，在线用 live 静止，要登录或配置用 needs，出错用 failed，停用和其余状态用 idle。 */
export function buddyTone(v: BuddyView): BuddyTone {
  if ((v.working ?? 0) > 0) return { tone: "live", still: false };
  if (v.builtin) return { tone: "live", still: true };
  if (installed(v) && !v.config.enabled) return { tone: "idle", still: true };
  switch (v.state) {
    case "online":
      return { tone: "live", still: true };
    case "needs_auth":
    case "needs_config":
      return { tone: "needs", still: true };
    case "error":
      return { tone: "failed", still: true };
    default:
      return { tone: "idle", still: true };
  }
}

/** buddyPrimary 按状态给出主操作。没装：能托管就「安装」，否则能装到系统就「装到系统」。不支持、检查中和内置的 yoyo 不放按钮。 */
export function buddyPrimary(v: BuddyView): BuddyAction {
  if (v.builtin) return "none";
  switch (v.state) {
    case "not_installed":
      if (v.canManage) return "install";
      return v.systemInstall?.available ? "installSystem" : "none";
    case "installing":
      return "cancel";
    case "needs_auth":
      return "login";
    case "needs_config":
      return "configure";
    case "offline":
    case "error":
    case "recently_lost":
      return "retry";
    case "online":
      return "test";
    default:
      return "none";
  }
}

/** updateTo 返回托管版能更新到的版本，没有更新时返回空串。只比数字段，比不出来就当没有。 */
export function updateTo(v: BuddyView): string {
  if (v.install.kind !== "managed" || !v.latest || !v.version) return "";
  return newer(v.latest, v.version) ? v.latest : "";
}

/** newer 判断版本 a 是否比 b 新。按点分的数字段逐段比，前缀 v 和预发布后缀不看。 */
export function newer(a: string, b: string): boolean {
  const parts = (s: string) => s.replace(/^v/i, "").split(/[-+]/)[0].split(".").map((x) => Number.parseInt(x, 10));
  const x = parts(a);
  const y = parts(b);
  if (x.some(Number.isNaN) || y.some(Number.isNaN)) return false;
  for (let i = 0; i < Math.max(x.length, y.length); i++) {
    const d = (x[i] ?? 0) - (y[i] ?? 0);
    if (d !== 0) return d > 0;
  }
  return false;
}

/** buddyMenu 列出更多菜单里该出现的项，顺序固定。主按钮已经是「装到系统」时菜单里不再重复。 */
export function buddyMenu(v: BuddyView): BuddyMenuItem[] {
  if (v.builtin) return [];
  const out: BuddyMenuItem[] = [];
  const has = installed(v);
  const installing = v.state === "installing";
  if (!has && !installing && v.systemInstall?.available && buddyPrimary(v) !== "installSystem") out.push("installSystem");
  if (!installing && v.install.kind === "managed" && v.system) out.push("useSystem");
  if (!installing && v.install.kind === "system" && v.canManage) out.push("useManaged");
  if (!installing && updateTo(v)) out.push("update");
  if (has && !installing) out.push("recheck");
  if (v.install.kind === "managed" && v.install.path) out.push("openDir");
  if (v.state !== "unsupported") out.push("logs");
  if (v.docs) out.push("docs");
  if (v.custom) out.push("remove");
  else if (has && !installing) out.push("uninstall");
  return out;
}

/**
 * actionLabel 是主按钮或菜单项上的字。在线时的主按钮用短的「测试」，菜单里的更新带上版本号。
 * 本机已经装着它的命令行时，装适配器这一步叫「连接」：人要的是连上已有的，不是再装一个。
 */
export function actionLabel(action: BuddyAction | BuddyMenuItem, v: BuddyView, c: BuddyCopy): string {
  switch (action) {
    case "none":
      return "";
    case "install":
      return foundOnHost(v) && v.host?.kind !== "app" ? c.action.connect : c.action.install;
    case "test":
      return c.action.testShort;
    case "update":
      return fill(c.action.update, { v: updateTo(v) });
    default:
      return c.action[action];
  }
}

/** sortBuddies 排序：内置的 yoyo 最前，然后按精选顺序，再按名字。 */
export function sortBuddies(list: BuddyView[]): BuddyView[] {
  return list.slice().sort((a, b) => {
    if (!!a.builtin !== !!b.builtin) return a.builtin ? -1 : 1;
    const oa = a.order || Number.MAX_SAFE_INTEGER;
    const ob = b.order || Number.MAX_SAFE_INTEGER;
    if (oa !== ob) return oa - ob;
    return a.name.localeCompare(b.name);
  });
}

/**
 * buddySections 把列表分成五组。
 * 在线：内置的 yoyo 和装好且没出问题的（包括检查中、刚断开）。
 * 需要处理：装好了但要登录、要配置、出错或离线的。
 * 在这台电脑上发现：本机有它的命令行或桌面应用，但还不能直接派的（见 foundOnHost）。
 * 可安装：精选里还没装的，以及正在安装的（安装中的项不能藏进折叠的「更多」）。
 * 更多：其余没装的，此系统不可用的排在最后。
 */
export function buddySections(list: BuddyView[]): BuddySections {
  const out: BuddySections = { online: [], needs: [], found: [], available: [], more: [] };
  const unsupported: BuddyView[] = [];
  for (const v of sortBuddies(list)) {
    if (v.builtin) out.online.push(v);
    else if (v.state === "installing") out.available.push(v);
    else if (installed(v)) (NEEDS_YOU.has(v.state) && (v.working ?? 0) === 0 ? out.needs : out.online).push(v);
    else if (v.state === "unsupported") unsupported.push(v);
    else if (foundOnHost(v)) out.found.push(v);
    else if (v.featured) out.available.push(v);
    else out.more.push(v);
  }
  out.more.push(...unsupported);
  return out;
}

/** hasExternal 表示装了至少一个外部 agent。 */
export function hasExternal(list: BuddyView[]): boolean {
  return list.some(installed);
}

/** phaseText 是安装阶段的说法，不认识的阶段说「安装中」。 */
export function phaseText(phase: string, c: BuddyCopy): string {
  return (c.phase as Record<string, string>)[phase] ?? c.state.installing;
}

/** progressText 是安装中的状态句：阶段，有百分比时带上百分比。 */
export function progressText(v: BuddyView, c: BuddyCopy): string {
  const p = v.progress;
  if (!p) return c.state.installing;
  const phase = phaseText(p.phase, c);
  return p.percent ? `${phase} · ${p.percent}%` : phase;
}

/** progressBar 决定进度细线怎么画。排队和装依赖时百分比会停很久，画成不确定态。 */
export function progressBar(v: BuddyView): BuddyBar {
  const p = v.progress;
  if (!p || !p.percent || p.phase === "queued" || p.phase === "deps") return { value: p?.percent ?? 0, indeterminate: true };
  return { value: p.percent, indeterminate: false };
}

/** buddyStatusText 是一行里的状态句。安装中讲阶段；在做任务讲个数；停用的说已停用；要配置、出错、离线、没装成和不支持时带上原因。 */
export function buddyStatusText(v: BuddyView, c: BuddyCopy): string {
  if (v.builtin) return (v.working ?? 0) > 0 ? fill(c.state.working, { n: v.working ?? 0 }) : c.builtinNote;
  if (v.state === "installing") return progressText(v, c);
  if ((v.working ?? 0) > 0) return fill(c.state.working, { n: v.working ?? 0 });
  if (installed(v) && !v.config.enabled) return c.disabled;
  if (foundOnHost(v)) return hostNote(v, c);
  const label = c.state[v.state] ?? v.state;
  const explains = NEEDS_YOU.has(v.state) || v.state === "unsupported" || v.state === "not_installed";
  const why = v.detail && v.detail !== label && explains ? v.detail : "";
  if (v.state === "unsupported") return why || label;
  return why ? `${label} · ${why}` : label;
}

/** setupSecrets 是一键配置里要列的密钥：必填的、能从 yoyo 复用的，以及已经存过的。 */
export function setupSecrets(v: BuddyView): BuddySecret[] {
  return v.config.secrets.filter((s) => !s.optional || s.reusable || s.hasKey);
}

/** splitLines 把多行文本拆成去掉首尾空白的非空行。 */
export function splitLines(text: string): string[] {
  return text.split(/\r?\n/).map((s) => s.trim()).filter(Boolean);
}

/** parseEnv 把「名字=值」每行一个的文本变成对象。有不合格的行时 bad 给出那一行。 */
export function parseEnv(text: string): { env: Record<string, string>; bad: string } {
  const env: Record<string, string> = {};
  for (const line of splitLines(text)) {
    const at = line.indexOf("=");
    const name = at > 0 ? line.slice(0, at).trim() : "";
    if (!/^[A-Za-z_][A-Za-z0-9_]*$/.test(name)) return { env: {}, bad: line };
    env[name] = line.slice(at + 1).trim();
  }
  return { env, bad: "" };
}

/** envText 把环境变量对象写回每行一个的文本，按名字排序。 */
export function envText(env?: Record<string, string>): string {
  return Object.keys(env ?? {})
    .sort()
    .map((k) => `${k}=${env?.[k] ?? ""}`)
    .join("\n");
}

/** formatSize 把字节数写成 B、KB、MB 或 GB，保留一位小数。 */
export function formatSize(bytes: number): string {
  if (!Number.isFinite(bytes) || bytes <= 0) return "0 B";
  const units = ["B", "KB", "MB", "GB", "TB"];
  let n = bytes;
  let i = 0;
  while (n >= 1024 && i < units.length - 1) {
    n /= 1024;
    i++;
  }
  return i === 0 ? `${n} B` : `${n.toFixed(n >= 100 ? 0 : 1)} ${units[i]}`;
}

/** checkedAgo 把检查时间写成「3 分钟前」这样的相对说法。now 和 locale 由调用方给，便于测试。读不出时间时返回空串。 */
export function checkedAgo(iso: string | undefined, now: number, locale: string): string {
  const at = iso ? Date.parse(iso) : Number.NaN;
  if (!Number.isFinite(at)) return "";
  const s = Math.round((at - now) / 1000);
  const fmt = new Intl.RelativeTimeFormat(locale, { numeric: "auto" });
  const abs = Math.abs(s);
  if (abs < 45) return fmt.format(0, "second");
  if (abs < 45 * 60) return fmt.format(Math.round(s / 60), "minute");
  if (abs < 22 * 3600) return fmt.format(Math.round(s / 3600), "hour");
  return fmt.format(Math.round(s / 86400), "day");
}
