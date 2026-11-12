/**
 * 拥有：截图用的一天示例数据：主会话的对话、编排台快照、待审队列、记忆、简报和 Buddy 名单，以及把它们接到假后端上的路由。
 * 不拥有：假后端本身（e2e/mock.ts 和 e2e/buddy-mock.ts）。这里只在它上面再挂几条路由，后挂的先命中。
 * 数据是示例，不是任何一台电脑上的真实数据。
 */

import type { Page } from "@playwright/test";
import { YOYO_BUDDY, buddyFixture } from "../e2e/buddy-mock";
import { mockApi } from "../e2e/mock";

export const WORKSPACE = "D:/work/yoyo";
export const MAIN = "main";

const now = Date.now();
const ago = (minutes: number) => new Date(now - minutes * 60_000).toISOString();
const today = () => {
  const d = new Date();
  const pad = (n: number) => String(n).padStart(2, "0");
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}`;
};

type Ev = Record<string, any>;

/** conversation 是主会话里的一天：一条简报、一件交出去的事、工人的回报、一个等回答的问题、一次检查点和一条已经停下的任务。 */
export function conversation(): Ev[] {
  let seq = 0;
  const ev = (type: string, source: string, minutes: number, payload: Ev): Ev => ({
    type, source, session_id: MAIN, ts: ago(minutes), seq: ++seq, payload,
  });
  /** 昨天下午的一个时刻。按日历日取，半夜跑截图时它也还是昨天。 */
  const yesterday = (hour: number, minute: number) => {
    const d = new Date(now);
    d.setDate(d.getDate() - 1);
    d.setHours(hour, minute, 0, 0);
    return d.toISOString();
  };
  const past = (type: string, source: string, ts: string, payload: Ev): Ev => ({ type, source, session_id: MAIN, ts, seq: ++seq, payload });
  const brief = {
    cards: [
      { id: "situation", layout: "checks", kicker: "今天先确认", title: "这 2 项一确认，进行中的工作就能继续。", lede: "到期 1 项，等人 2 项，卡住 0 项。其余先不动。", items: [{ title: "起草本周周报", detail: "第 3 节要不要写 Harbor 的数据？" }, { title: "整理明早评审材料", detail: "做好了，等你看一眼" }], choices: [], primary: "", secondary: "" },
    ],
    picked: {},
    woven: true,
  };
  return [
    past("user", "user", yesterday(16, 20), { id: "u0", text: "明早评审要用的材料帮我整理一下，放到桌面的「评审」文件夹。" }),
    past("assistant", "agent", yesterday(16, 20), { id: "main:r0", text: "交给「整理明早评审材料」了，整理好放到桌面的「评审」文件夹。" }),
    past("turn_end", "runtime", yesterday(16, 21), { id: "e0" }),
    ev("assistant", "desk", 190, { id: `brief:morning:${today()}`, kind: "brief", slot: "morning", day: today(), chip: "晨间简报", brief: JSON.stringify(brief), text: "早上好。今天有两件事等你确认，一件在跑。" }),
    ev("user", "user", 170, { id: "u1", text: "把 #162 的 CI 失败修一下，改完先别推。另外记住：报告结论先行。" }),
    ev("tool_call", "agent", 170, { id: "c1", name: "memory_write", arguments: JSON.stringify({ kind: "semantic", text: "报告结论先行" }) }),
    ev("tool_result", "agent", 170, { id: "c1", name: "memory_write", content: "记下了 mem-12：报告结论先行" }),
    ev("tool_call", "agent", 170, { id: "c2", name: "task_route", arguments: JSON.stringify({ items: [{ trigger: "desk:main:t1", action: "accept", title: "修复 #162 CI 失败" }] }) }),
    ev("tool_result", "agent", 170, { id: "c2", name: "task_route", content: "accepted desk:main:t1" }),
    ev("assistant", "agent", 169, { id: "main:r1", text: "已经交给「修复 #162 CI 失败」这件任务，这一轮先找到失败原因并在本地改好，不推送。另外记下了：报告结论先行。" }),
    ev("turn_end", "runtime", 169, { id: "e1" }),
    ev("assistant", "worker", 120, { id: "handoff:tkt-12:a", kind: "status", ticket: "tkt-12", state: "ready", agent: "codex", agentName: "Codex", text: "失败来自 `desk_push.go` 里同一天重复推送的判断：时区按 UTC 计算，北京时间早上 8 点前的推送会被算成前一天。改成按本地时区取日期，补了测试，本地通过。改动还没推送。" }),
    ev("compaction", "runtime", 100, { kind: "checkpoint", note: "Checkpoint · kept 16212 · elided 42 · hydrated 0 files · budget", kept: 16212, elided: 42, hydrated: 0, phase: "complete" }),
    ev("user", "user", 62, { id: "u2", text: "周报先起个草，按上周的格式。" }),
    ev("tool_call", "agent", 62, { id: "c3", name: "task_route", arguments: JSON.stringify({ items: [{ trigger: "desk:main:t2", action: "accept", title: "起草本周周报" }] }) }),
    ev("tool_result", "agent", 62, { id: "c3", name: "task_route", content: "accepted desk:main:t2" }),
    ev("assistant", "agent", 61, { id: "main:r2", text: "交给「起草本周周报」了，按上周的格式先出一版草稿。" }),
    ev("turn_end", "runtime", 61, { id: "e2" }),
    ev("assistant", "worker", 35, { id: "handoff:tkt-14:b", kind: "handoff", ticket: "tkt-14", state: "question", text: "周报第 3 节要不要写 Harbor 的数据？写的话我按上周的格式补一段。" }),
    ev("assistant", "worker", 12, { id: "handoff:tkt-9:c", kind: "status", ticket: "tkt-9", state: "stopped", text: "回复 #158 的评论已经停下：作者说先不改。" }),
  ];
}

function ticket(partial: Ev & { id: string }): Ev {
  return { stage: "working", action: "running", person: false, origin: "desk", openedAt: ago(170), updatedAt: ago(5), ...partial };
}

/** deskView 是编排台快照：两件等你、一件在做、两件结束，外加一条等接受的来信和一项今天到期的日程。 */
export function deskView(): Ev {
  return {
    mode: "observe",
    paused: false,
    admission: { listedUnaddressed: "explicit", unlistedAddressed: "explicit", unlistedUnaddressed: "ignore" },
    sources: [],
    tickets: [
      ticket({ id: "tkt-14", title: "起草本周周报", stage: "person", action: "answer", person: true, attention: "第 3 节要不要写 Harbor 的数据？", updatedAt: ago(35), assignments: [{ id: "a14", state: "question", lines: [{ state: "question", question: "第 3 节要不要写 Harbor 的数据？", at: ago(35) }] }] }),
      ticket({ id: "tkt-13", title: "整理明早评审材料", stage: "person", action: "review", person: true, updatedAt: ago(48), assignments: [{ id: "a13", state: "ready", lines: [{ state: "ready", summary: "做好了，等你看一眼", at: ago(48) }] }] }),
      ticket({ id: "tkt-12", title: "修复 #162 CI 失败", agent: "codex", updatedAt: ago(120), assignments: [{ id: "a12", state: "progress", agent: "codex", lines: [{ state: "progress", summary: "本地测试通过，改动还没推送", at: ago(120) }] }] }),
      ticket({ id: "tkt-9", title: "回复 #158 的评论", stage: "ended", action: "", outcome: "stopped", updatedAt: ago(12) }),
      ticket({ id: "tkt-8", title: "把发布说明翻成英文", stage: "ended", action: "", outcome: "completed", updatedAt: ago(400) }),
    ],
    threads: [],
    queued: [
      { id: "gh-163", source: "github", sender: "alice", text: "@yoyo 帮忙看一下 #163 的构建为什么在 Windows 上挂了", addressed: true, at: ago(130), queued: true },
      { id: "cal-1", source: "calendar", sender: "calendar", text: "产品周会", addressed: false, at: ago(600), due: today(), queued: true },
    ],
    evals: [],
    decisions: [],
    problems: [],
    rules: [{ id: "daily-brief", text: "每天早上整理今天要确认的事", enabled: true }],
    health: "ok",
    at: new Date().toISOString(),
    groups: [],
  };
}

/** memoryRows 是记忆页的示例：本机清点、你说过的话、偏好和项目。 */
export function memoryRows(): Ev[] {
  return [
    { id: "machine:device", kind: "profile", text: "ThinkPad X1 Carbon", source: "machine", updated_at: ago(720) },
    { id: "machine:host", kind: "profile", text: "SCX-X1", source: "machine", updated_at: ago(720) },
    { id: "machine:os", kind: "profile", text: "Windows 11 企业版", source: "machine", updated_at: ago(720) },
    { id: "machine:zone", kind: "profile", text: "Asia/Shanghai", source: "machine", updated_at: ago(720) },
    { id: "machine:git", kind: "profile", text: "Shenchangxin", source: "machine", updated_at: ago(720) },
    { id: "mem-3", kind: "profile", text: "在做一个本地优先的桌面 Agent", source: "User confirmed in chat", created_at: ago(4000), updated_at: ago(4000) },
    { id: "mem-12", kind: "semantic", text: "报告结论先行", source: "User confirmed in chat", created_at: ago(170), updated_at: ago(170), origin: `session:${MAIN}` },
    { id: "mem-7", kind: "semantic", text: "回复用中文", source: "User confirmed in chat", created_at: ago(9000), updated_at: ago(9000) },
    { id: "mem-9", kind: "semantic", text: "周末不要推送工作提醒", source: "User confirmed in chat", created_at: ago(6000), updated_at: ago(6000) },
    { id: "line:yoyo", kind: "project", text: "yoyo", project: "yoyo", state: "moving", evidence: "最近 2 小时有改动", source: "machine", updated_at: ago(120) },
    { id: "line:hypit", kind: "project", text: "hypit", project: "hypit", state: "stalled", evidence: "停在空文件 notes.md", when: "3 天前", source: "machine", updated_at: ago(4300) },
  ];
}

/**
 * buddies 是 Buddy 名单的示例：一个要登录、一个要配置；两个在这台电脑上发现的（桌面应用里带着命令行的、只装了桌面应用的）；
 * 三个在线，其中一个是在 PATH 之外找到、直接连上的；还有能装的和收在「更多」里的。
 */
export function buddies(): Ev[] {
  const login = { methods: [{ id: "kimi-login", name: "用 Kimi 账号登录", type: "agent" }] };
  return [
    YOYO_BUDDY,
    buddyFixture("claude", {
      name: "Claude Code", vendor: "Anthropic", order: 1, cliOnly: true, description: "Anthropic 的编码 agent。",
      host: { kind: "cli", path: "C:/Users/me/AppData/Roaming/Claude/claude-code/2.1.10/claude.exe", app: "Claude" },
    }),
    buddyFixture("codex", { name: "Codex", vendor: "OpenAI", order: 2, install: { kind: "system", source: "path", path: "C:/Users/me/AppData/Roaming/npm/codex-acp.cmd" }, version: "0.4.1", state: "online", working: 1, checkedAt: ago(8), description: "OpenAI 的编码 agent。" }),
    buddyFixture("gemini", { name: "Gemini CLI", vendor: "Google", order: 3, install: { kind: "managed", source: "npm", path: "C:/Users/me/.yoyo/buddies/gemini" }, version: "0.9.0", state: "online", checkedAt: ago(8) }),
    buddyFixture("qwen", { name: "Qwen Code", vendor: "Alibaba", order: 4, install: { kind: "managed", source: "npm", path: "C:/Users/me/.yoyo/buddies/qwen" }, version: "0.6.2", state: "needs_config", detail: "DASHSCOPE_API_KEY", config: { secrets: [{ name: "DASHSCOPE_API_KEY", label: "DashScope API key", hasKey: false }] } }),
    buddyFixture("kimi", { name: "Kimi Code", vendor: "Moonshot AI", order: 5, install: { kind: "managed", source: "npm", path: "C:/Users/me/.yoyo/buddies/kimi" }, version: "0.31.0", state: "needs_auth", auth: login }),
    buddyFixture("cursor", {
      name: "Cursor", vendor: "Cursor", order: 11, canManage: false,
      host: { kind: "app", path: "C:/Users/me/AppData/Local/Programs/cursor/Cursor.exe", app: "Cursor" },
      systemInstall: { available: true, command: "powershell -NoProfile -ExecutionPolicy Bypass -Command \"irm 'https://cursor.com/install?win32=true' | iex\"" },
    }),
    buddyFixture("opencode", {
      name: "OpenCode", vendor: "OpenCode", order: 12, install: { kind: "system", source: "path", path: "C:/Users/me/.opencode/bin/opencode.exe" }, version: "1.4.2", state: "online", checkedAt: ago(8),
      host: { kind: "acp", path: "C:/Users/me/.opencode/bin/opencode.exe" },
    }),
    buddyFixture("copilot", { name: "GitHub Copilot", vendor: "GitHub", order: 7 }),
    buddyFixture("goose", { name: "goose", vendor: "Block", order: 13, featured: false }),
  ];
}

/** mockDay 把一天的示例数据接到页面上。要在页面打开前调用。over 可以换掉会话事件或 Buddy 名单。 */
export async function mockDay(page: Page, over: { events?: Ev[]; buddies?: Ev[]; desk?: Ev; locale?: string; context?: Ev } = {}) {
  const events = over.events ?? conversation();
  await mockApi(page, WORKSPACE, {
    sessions: [{ id: MAIN, title: "主会话", workspace: WORKSPACE, channel: "agent", auth_mode: "default", created_at: ago(9000) }],
    events,
    buddies: over.buddies ?? buddies(),
    config: { locale: over.locale ?? "zh-CN", model: "deepseek-chat", provider: "deepseek", models: ["deepseek-chat", "deepseek-reasoner"], show_thinking: false },
    context: over.context ?? { tokens: 18_400, budget: 108_000, window: 128_000, prefix_tokens: 5_200, dynamic_tokens: 900, schema_tokens: 2_100 },
    queue: {
      offers: [],
      inbox: [],
      drafts: [{ id: "d1", to: "王老师", subject: "评审材料今晚发", at: ago(125) }],
      proposals: [],
      waiting: [],
    },
  });
  const desk = over.desk ?? deskView();
  await page.route((url) => url.pathname.startsWith("/api/orchestrator"), async (route) => {
    const url = new URL(route.request().url());
    if (url.pathname === "/api/orchestrator") return route.fulfill({ json: desk });
    if (url.pathname === "/api/orchestrator/transcript") return route.fulfill({ json: [] });
    return route.fulfill({ json: { ok: true } });
  });
  await page.route((url) => url.pathname === "/api/memory", (route) => route.fulfill({ json: route.request().method() === "GET" ? memoryRows() : { ok: true } }));
  await page.route((url) => url.pathname === "/api/briefs", (route) => {
    const first = events.find((ev) => ev.payload?.kind === "brief");
    if (!first) return route.fulfill({ json: [] });
    return route.fulfill({ json: [{ id: first.payload.id, slot: first.payload.slot, day: first.payload.day, chip: first.payload.chip, text: first.payload.text, brief: first.payload.brief, sessionId: MAIN, at: first.ts }] });
  });
  await page.route((url) => url.pathname === "/api/desk/state", (route) => route.fulfill({ json: { briefing: { day: today(), speak: false, signals: [], open: [] }, mutes: [], at: new Date().toISOString() } }));
  await page.route((url) => url.pathname === "/api/poi", (route) => route.fulfill({ json: { points: [], added: [], dropped: [], sig: "", pending: false } }));
}
