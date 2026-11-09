import assert from "node:assert/strict";
import { describe, it } from "node:test";
import type { BuddyView } from "../../lib/client.ts";
import { en } from "../../lib/copy.ts";
import {
  agentStateText,
  attentionCount,
  blockedHint,
  buddyMenu,
  buddyPrimary,
  buddySections,
  buddyStatusText,
  buddyTone,
  canDispatch,
  checkedAgo,
  fill,
  formatSize,
  mentionAgents,
  newer,
  onlineCount,
  parseEnv,
  progressBar,
  progressText,
  routable,
  updateTo,
} from "./buddy-model.ts";

/**
 * 拥有：Buddy 栏目的纯判断：色调、主按钮、更多菜单、分组、角标、状态句、进度、表单解析，以及输入框 @ 里的 agent 列表。
 * 不拥有：组件怎么画，也不碰网络。
 */

const c = en.buddy;

/** view 造一项，默认是装好、启用、在线的托管 agent。 */
function view(patch: Partial<BuddyView> & { id: string }): BuddyView {
  return {
    name: patch.id,
    install: { kind: "managed", source: "npm", path: `/buddy/${patch.id}` },
    state: "online",
    config: { enabled: true, mode: "managed", approval: "yoyo", maxParallel: 2, secrets: [] },
    ...patch,
  };
}

const yoyo = view({ id: "yoyo", name: "yoyo", builtin: true, install: { kind: "none" } });
const notInstalled = (id: string, extra: Partial<BuddyView> = {}) => view({ id, install: { kind: "none" }, state: "not_installed", canManage: true, ...extra });

describe("buddyTone", () => {
  /** 在做任务时呼吸，在线时静止，要登录和要配置是 needs，出错是 failed，停用和离线是 idle。 */
  it("maps states to status dots", () => {
    assert.deepEqual(buddyTone(view({ id: "a", working: 1 })), { tone: "live", still: false });
    assert.deepEqual(buddyTone(view({ id: "a" })), { tone: "live", still: true });
    assert.deepEqual(buddyTone(yoyo), { tone: "live", still: true });
    assert.equal(buddyTone(view({ id: "a", state: "needs_auth" })).tone, "needs");
    assert.equal(buddyTone(view({ id: "a", state: "needs_config" })).tone, "needs");
    assert.equal(buddyTone(view({ id: "a", state: "error" })).tone, "failed");
    assert.equal(buddyTone(view({ id: "a", state: "offline" })).tone, "idle");
    assert.equal(buddyTone(view({ id: "a", config: { ...view({ id: "a" }).config, enabled: false } })).tone, "idle");
  });
});

describe("buddyPrimary", () => {
  /** 主按钮跟着状态走；不能托管时退到装到系统；不支持和内置的 yoyo 不放按钮。 */
  it("picks one action per state", () => {
    assert.equal(buddyPrimary(notInstalled("a")), "install");
    assert.equal(buddyPrimary(notInstalled("a", { canManage: false, systemInstall: { available: true, command: "npm i -g a" } })), "installSystem");
    assert.equal(buddyPrimary(notInstalled("a", { canManage: false })), "none");
    assert.equal(buddyPrimary(view({ id: "a", state: "installing" })), "cancel");
    assert.equal(buddyPrimary(view({ id: "a", state: "needs_auth" })), "login");
    assert.equal(buddyPrimary(view({ id: "a", state: "needs_config" })), "configure");
    assert.equal(buddyPrimary(view({ id: "a", state: "offline" })), "retry");
    assert.equal(buddyPrimary(view({ id: "a", state: "error" })), "retry");
    assert.equal(buddyPrimary(view({ id: "a", state: "online" })), "test");
    assert.equal(buddyPrimary(view({ id: "a", state: "unsupported" })), "none");
    assert.equal(buddyPrimary(yoyo), "none");
  });
});

describe("buddySections", () => {
  /** yoyo 排第一；有问题的进需要处理；精选和正在装的进可安装；其余进更多，不支持的垫底。 */
  it("groups agents the way the rail shows them", () => {
    const s = buddySections([
      notInstalled("zed-x", { state: "unsupported", canManage: false }),
      notInstalled("other"),
      view({ id: "codex", order: 2 }),
      view({ id: "gemini", state: "needs_auth" }),
      notInstalled("claude", { featured: true, order: 1 }),
      notInstalled("slow", { state: "installing" }),
      view({ id: "lost", state: "recently_lost" }),
      view({ id: "broken", state: "error" }),
      yoyo,
    ]);
    assert.deepEqual(s.online.map((v) => v.id), ["yoyo", "codex", "lost"]);
    assert.deepEqual(s.needs.map((v) => v.id), ["broken", "gemini"]);
    assert.deepEqual(s.available.map((v) => v.id), ["claude", "slow"]);
    assert.deepEqual(s.more.map((v) => v.id), ["other", "zed-x"]);
  });

  /** 角标只数启用着却接不了任务的；在线个数把 yoyo 也算上。 */
  it("counts the dock badge and online agents", () => {
    const off = { ...view({ id: "x" }).config, enabled: false };
    const list = [yoyo, view({ id: "a" }), view({ id: "b", state: "needs_auth" }), view({ id: "c", state: "offline", config: off }), notInstalled("d")];
    assert.equal(attentionCount(list), 1);
    assert.equal(onlineCount(list), 2);
    assert.equal(canDispatch(view({ id: "e", config: off })), false);
    assert.equal(canDispatch(yoyo), true);
  });
});

describe("buddyMenu", () => {
  /** 托管版旁边有系统版时能切换；有新版本时能更新；自定义的只能移除。 */
  it("lists only actions that apply", () => {
    const managed = view({ id: "a", system: { path: "/usr/bin/a" }, version: "1.2.0", latest: "1.10.0", docs: "https://x" });
    assert.deepEqual(buddyMenu(managed), ["useSystem", "update", "recheck", "openDir", "logs", "docs", "uninstall"]);
    const system = view({ id: "b", install: { kind: "system", source: "path", path: "/usr/bin/b" }, canManage: true });
    assert.deepEqual(buddyMenu(system), ["useManaged", "recheck", "logs", "uninstall"]);
    const custom = view({ id: "c", custom: true, install: { kind: "custom", source: "path", path: "/opt/c" } });
    assert.deepEqual(buddyMenu(custom), ["recheck", "logs", "remove"]);
    assert.deepEqual(buddyMenu(notInstalled("d", { systemInstall: { available: true, command: "npm i -g d" } })), ["installSystem", "logs"]);
    assert.deepEqual(buddyMenu(yoyo), []);
  });

  /** 版本按数字段比较，比不出来就不提示更新。 */
  it("compares versions by number", () => {
    assert.equal(newer("1.10.0", "1.9.9"), true);
    assert.equal(newer("v2.0.0", "2.0.0"), false);
    assert.equal(newer("2.0.0-beta.1", "1.9.0"), true);
    assert.equal(newer("nightly", "1.0.0"), false);
    assert.equal(updateTo(view({ id: "a", version: "1.0.0", latest: "1.0.1" })), "1.0.1");
    assert.equal(updateTo(view({ id: "a", install: { kind: "system", source: "path" }, version: "1.0.0", latest: "1.0.1" })), "");
  });
});

describe("status text", () => {
  /** 要配置时带上缺的名字；没装成时带上原因；停用、只找到命令、在做任务各有说法；不支持只说原因。 */
  it("explains each row in one line", () => {
    assert.equal(buddyStatusText(view({ id: "a", state: "needs_config", detail: "OPENAI_API_KEY" }), c), "Needs setup · OPENAI_API_KEY");
    assert.equal(buddyStatusText(notInstalled("a", { detail: "Install failed: EACCES" }), c), "Not installed · Install failed: EACCES");
    assert.equal(buddyStatusText(notInstalled("a"), c), "Not installed");
    assert.equal(buddyStatusText(view({ id: "a", working: 2 }), c), "Working on 2");
    assert.equal(buddyStatusText(view({ id: "a", config: { ...view({ id: "a" }).config, enabled: false } }), c), c.disabled);
    assert.equal(buddyStatusText(notInstalled("a", { cliOnly: true }), c), c.cliOnly);
    assert.equal(buddyStatusText(notInstalled("a", { state: "unsupported", detail: "No build for this system" }), c), "No build for this system");
    assert.equal(buddyStatusText(yoyo, c), c.builtinNote);
  });

  /** 装依赖和排队时画不确定态；有百分比的阶段画确定进度并写进状态句。 */
  it("describes install progress", () => {
    const deps = view({ id: "a", state: "installing", progress: { phase: "deps", percent: 40 } });
    assert.deepEqual(progressBar(deps), { value: 40, indeterminate: true });
    assert.equal(progressText(deps, c), "Installing dependencies · 40%");
    const dl = view({ id: "a", state: "installing", progress: { phase: "download", percent: 55 } });
    assert.deepEqual(progressBar(dl), { value: 55, indeterminate: false });
    assert.equal(progressText(view({ id: "a", state: "installing", progress: { phase: "node" } }), c), "Preparing Node runtime");
    assert.equal(progressText(view({ id: "a", state: "installing", progress: { phase: "mystery" } }), c), c.state.installing);
  });
});

describe("helpers", () => {
  /** 环境变量每行一个，名字不合格时指出那一行。 */
  it("parses env lines", () => {
    assert.deepEqual(parseEnv("A=1\n\n B_2 = two words \n"), { env: { A: "1", B_2: "two words" }, bad: "" });
    assert.deepEqual(parseEnv("A=1\nnot a pair"), { env: {}, bad: "not a pair" });
    assert.deepEqual(parseEnv("=x"), { env: {}, bad: "=x" });
  });

  /** 占位按名字替换，没给的留着；字节数换成合适的单位；检查时间写成相对说法。 */
  it("formats text, sizes and times", () => {
    assert.equal(fill("{name} is ready, {n} left, {missing}", { name: "Codex", n: 2 }), "Codex is ready, 2 left, {missing}");
    assert.equal(formatSize(0), "0 B");
    assert.equal(formatSize(512), "512 B");
    assert.equal(formatSize(1536), "1.5 KB");
    assert.equal(formatSize(250 * 1024 * 1024), "250 MB");
    const now = Date.parse("2026-11-09T08:00:00Z");
    assert.equal(checkedAgo("2026-11-09T07:57:00Z", now, "en"), "3 minutes ago");
    assert.equal(checkedAgo("2026-11-09T07:59:50Z", now, "en"), "now");
    assert.equal(checkedAgo("", now, "en"), "");
  });
});

describe("agent mentions", () => {
  /** @ 菜单只在装了外部 agent 时有。yoyo 在前，能派的排在前面；编号、名字、厂商和中文别名都能搜到。 */
  it("lists installed agents, routable first, matched by alias", () => {
    const codex = view({ id: "codex", name: "Codex", vendor: "OpenAI", order: 2 });
    const claude = view({ id: "claude", name: "Claude Code", vendor: "Anthropic", order: 1, state: "needs_auth", aliases: ["克劳德"] });
    const gemini = notInstalled("gemini");
    assert.deepEqual(mentionAgents([yoyo, gemini], ""), []);
    assert.deepEqual(mentionAgents([gemini, claude, codex, yoyo], "").map((v) => v.id), ["yoyo", "codex", "claude"]);
    assert.deepEqual(mentionAgents([yoyo, claude, codex], "克劳").map((v) => v.id), ["claude"]);
    assert.deepEqual(mentionAgents([yoyo, claude, codex], "openai").map((v) => v.id), ["codex"]);
  });

  /** 能不能派和路由一致：检查中、刚断开也算，停用、要登录、离线不算。点了派不了的，说清先做什么。 */
  it("routes like the router and explains the blocked ones", () => {
    const off = view({ id: "a", name: "A", config: { ...view({ id: "a" }).config, enabled: false } });
    assert.equal(routable(view({ id: "a", state: "probing" })), true);
    assert.equal(routable(view({ id: "a", state: "recently_lost" })), true);
    assert.equal(routable(view({ id: "a", state: "offline" })), false);
    assert.equal(routable(off), false);
    assert.equal(routable(yoyo), true);
    assert.equal(blockedHint(off, en.composer), "Turn on A in Buddy first");
    assert.equal(blockedHint(view({ id: "c", name: "Claude Code", state: "needs_auth" }), en.composer), "Sign in to Claude Code first");
    assert.equal(blockedHint(view({ id: "g", name: "Gemini CLI", state: "needs_config" }), en.composer), "Set up Gemini CLI first");
    assert.equal(blockedHint(view({ id: "q", name: "Qwen", state: "offline" }), en.composer), "Qwen can't take tasks right now");
    assert.equal(agentStateText(view({ id: "a", working: 2 }), c), "Working on 2");
    assert.equal(agentStateText(off, c), c.disabled);
    assert.equal(agentStateText(yoyo, c), c.state.online);
  });
});
