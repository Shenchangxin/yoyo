/**
 * 拥有：多 agent 的端到端用例。任务栏：任务组一行的展开收起和依赖提示、分工页的车道和动态、成员工单上回到任务组的面包屑、
 * 停止全部要点两次、对比页的采用，以及任务组放到中间。主会话：点两个 agent 发出去后，路由的回复下面跟着任务组卡片，
 * 成员状态折进卡片，成员提问带身份和面包屑，唤醒回执，结束后卡片收成一行；对比收起后写路由推荐，采用后写已采用。
 * 不拥有：任务组怎么拆、怎么放行（Go 的 group_test.go）。假后端在 mock.ts 和 buddy-mock.ts，编排台快照由这里的 mockDesk 给出，
 * 会话的实时尾部由 mockLive 给出。每条 test 的标题就是它断言什么。
 */

import { expect, test, type Page } from "@playwright/test";
import { YOYO_BUDDY, buddyFixture } from "./buddy-mock";
import { mockApi } from "./mock";

const CODEX = buddyFixture("codex", {
  name: "Codex",
  vendor: "OpenAI",
  order: 2,
  install: { kind: "system", path: "C:/Users/me/AppData/Roaming/npm/codex-acp.cmd" },
  state: "online",
});
const BUDDIES = [YOYO_BUDDY, CODEX];
const SESSIONS = [{ id: "s1", title: "Demo thread", workspace: "C:/tmp/ws" }];

function minutesAgo(n: number): string {
  return new Date(Date.now() - n * 60_000).toISOString();
}

/** ticket 是快照里的一张工单，没给的字段按还在做补齐。 */
function ticket(partial: Record<string, any> & { id: string }): Record<string, any> {
  return { stage: "working", action: "running", person: false, origin: "desk", title: partial.id, openedAt: minutesAgo(30), updatedAt: minutesAgo(5), ...partial };
}

/** deskView 是编排台快照，只填任务栏用得到的部分。 */
function deskView(tickets: Record<string, any>[], groups: Record<string, any>[]): Record<string, any> {
  return {
    mode: "auto",
    paused: false,
    admission: { listedUnaddressed: "explicit", unlistedAddressed: "explicit", unlistedUnaddressed: "ignore" },
    sources: [],
    tickets,
    threads: [],
    queued: [],
    evals: [],
    decisions: [],
    problems: [],
    rules: [],
    health: "ok",
    at: new Date().toISOString(),
    groups,
  };
}

type Desk = { view: Record<string, any>; acts: Record<string, string>[]; transcripts: Record<string, unknown[]> };

/** mockDesk 接管 /api/orchestrator：快照每次现读 desk.view，动作记进 desk.acts，onAct 可以就地改快照。要挂在 mockApi 之后，后挂的路由先命中。 */
async function mockDesk(page: Page, desk: Desk, onAct?: (body: Record<string, string>) => void) {
  await page.route((url) => url.pathname.startsWith("/api/orchestrator"), async (route) => {
    const url = new URL(route.request().url());
    if (url.pathname === "/api/orchestrator") return route.fulfill({ json: desk.view });
    if (url.pathname === "/api/orchestrator/transcript") return route.fulfill({ json: desk.transcripts[url.searchParams.get("ticket") || ""] ?? [] });
    if (url.pathname === "/api/orchestrator/act") {
      const body = route.request().postDataJSON() as Record<string, string>;
      desk.acts.push(body);
      onAct?.(body);
      return route.fulfill({ json: { ok: true } });
    }
    return route.fulfill({ json: {} });
  });
}

/** splitDesk 是一组分工：后端交给 Codex，正在问端口；前端交给 yoyo，挂着等后端。另有一张不在组里的工单。 */
function splitDesk(): Desk {
  const back = ticket({
    id: "tkt-1",
    title: "Write the API",
    role: "Backend",
    agent: "codex",
    group: "grp-1",
    stage: "person",
    action: "answer",
    person: true,
    attention: "Which port should the API use?",
    directory: "C:/tmp/ws/.yoyo/tickets/tkt-1",
    assignments: [{
      id: "asg-1",
      state: "question",
      agent: "codex",
      lines: [
        { state: "progress", summary: "Created the tables", at: minutesAgo(20) },
        { state: "question", question: "Which port should the API use?", at: minutesAgo(10) },
      ],
    }],
  });
  const front = ticket({ id: "tkt-2", title: "Build the page", role: "Frontend", agent: "yoyo", group: "grp-1", action: "queued", held: "deps", dependsOn: ["tkt-1"] });
  const solo = ticket({ id: "tkt-3", title: "Write the weekly report" });
  const group = {
    id: "grp-1",
    mode: "split",
    title: "Build the todo app",
    session: "s1",
    tickets: ["tkt-1", "tkt-2"],
    state: "attention",
    done: 0,
    total: 2,
    sharedDir: "C:/tmp/ws/.yoyo/groups/grp-1/shared",
    openedAt: minutesAgo(30),
    updatedAt: minutesAgo(10),
  };
  return {
    view: deskView([back, front, solo], [group]),
    acts: [],
    transcripts: {
      "tkt-1": [
        { id: "r1", role: "tool_result", name: "write_file", text: "wrote server/api.go" },
        { id: "r2", role: "tool_result", name: "edit_file", text: "edited server/db.go" },
      ],
    },
  };
}

/** compareDesk 是一组对比：Codex 和 yoyo 各修了一遍登录，都做完了，还没采用。 */
function compareDesk(): Desk {
  const done = (id: string, agent: string, summary: string) =>
    ticket({
      id,
      title: "Fix the login bug",
      agent,
      group: "grp-2",
      stage: "ended",
      action: "",
      outcome: "completed",
      directory: `C:/tmp/ws/.yoyo/tickets/${id}`,
      assignments: [{ id: `asg-${id}`, state: "ready", agent, lines: [{ state: "ready", summary, at: minutesAgo(3) }] }],
    });
  const group = {
    id: "grp-2",
    mode: "compare",
    title: "Fix the login bug",
    session: "s1",
    tickets: ["tkt-4", "tkt-5"],
    state: "done",
    done: 2,
    total: 2,
    openedAt: minutesAgo(40),
    updatedAt: minutesAgo(3),
  };
  return {
    view: deskView([done("tkt-4", "codex", "Fixed the token refresh"), done("tkt-5", "yoyo", "Rewrote the session store")], [group]),
    acts: [],
    transcripts: { "tkt-4": [{ id: "r1", role: "tool_result", name: "edit_file", text: "edited auth/refresh.go" }] },
  };
}

/** openTasks 打开首页，等今天栏出来，切到任务。 */
async function openTasks(page: Page) {
  await page.goto("/");
  const tab = page.getByTestId("work-tab-tasks");
  await expect(tab).toBeVisible({ timeout: 15_000 });
  await tab.click();
}

/** actSent 等下一次编排台动作，返回它的请求体。 */
function actSent(page: Page) {
  return page.waitForRequest((r) => r.method() === "POST" && new URL(r.url()).pathname === "/api/orchestrator/act");
}

test("a split group folds its members under one row with who does what and who waits on whom", async ({ page }) => {
  await mockApi(page, "C:/tmp/ws", { sessions: SESSIONS, buddies: BUDDIES });
  await mockDesk(page, splitDesk());
  await openTasks(page);
  const row = page.getByTestId("group-row");
  await expect(row).toHaveCount(1, { timeout: 10_000 });
  await expect(row.getByTestId("group-open")).toContainText("Build the todo app");
  await expect(row.getByTestId("group-progress")).toHaveText("Split · 0/2");
  const toggle = row.getByTestId("group-toggle");
  await expect(toggle).toHaveAttribute("aria-expanded", "true");
  const members = row.getByTestId("task-row");
  await expect(members).toHaveCount(2);
  await expect(members.nth(0).getByTestId("task-row-agent")).toHaveText("Backend · Codex", { timeout: 10_000 });
  await expect(members.nth(1).getByTestId("task-row-agent")).toHaveText("Frontend · yoyo");
  await expect(members.nth(1).getByTestId("task-hint")).toHaveText("Waiting on Backend");
  await expect(page.getByTestId("task-row")).toHaveCount(3);
  await expect(page.getByTestId("task-row").filter({ hasText: "Write the weekly report" }).getByTestId("task-row-agent")).toHaveCount(0);
  await toggle.click();
  await expect(toggle).toHaveAttribute("aria-expanded", "false");
  await expect(row.getByTestId("group-members")).toHaveCount(0);
  await toggle.click();
  await expect(row.getByTestId("task-row")).toHaveCount(2);
});

test("the split page lays members out by dependency, merges their reports, and a member page leads back to it", async ({ page }) => {
  await mockApi(page, "C:/tmp/ws", { sessions: SESSIONS, buddies: BUDDIES });
  await mockDesk(page, splitDesk());
  await openTasks(page);
  await page.getByTestId("group-open").click();
  const pane = page.getByTestId("group-page");
  await expect(pane.getByTestId("group-title")).toHaveText("Build the todo app");
  await expect(pane.getByTestId("group-status")).toHaveText("Split · Needs you · 0/2");
  const lanes = pane.getByTestId("group-lane");
  await expect(lanes).toHaveCount(2);
  await expect(lanes.nth(0)).toHaveAttribute("data-ticket", "tkt-1");
  await expect(lanes.nth(1).getByTestId("group-lane-status")).toHaveText("Waiting on Backend");
  await expect(lanes.nth(0)).toContainText("2 files changed", { timeout: 10_000 });
  await expect(pane.getByTestId("group-shared")).toContainText("C:/tmp/ws/.yoyo/groups/grp-1/shared");
  const events = pane.getByTestId("group-event");
  await expect(events).toHaveCount(2);
  await expect(events.nth(0)).toContainText("Which port should the API use?");
  await expect(events.nth(1)).toContainText("Created the tables");
  await expect(events.nth(1).getByTestId("group-answer")).toHaveCount(0);
  await events.nth(0).getByTestId("group-answer").click();
  const read = page.getByTestId("task-read");
  await expect(read).toBeVisible();
  const crumb = read.getByTestId("task-group-crumb");
  await expect(crumb).toHaveText("Task group: Build the todo app · Backend");
  await crumb.click();
  await expect(page.getByTestId("group-page")).toBeVisible();
  await page.getByTestId("group-page").getByRole("button", { name: "Back", exact: true }).click();
  await expect(page.getByTestId("group-row")).toBeVisible();
});

test("stop all needs a second click and then stops the whole group", async ({ page }) => {
  const desk = splitDesk();
  await mockApi(page, "C:/tmp/ws", { sessions: SESSIONS, buddies: BUDDIES });
  await mockDesk(page, desk, (body) => {
    if (body.kind !== "stopGroup") return;
    desk.view.groups[0].state = "stopped";
    for (const row of desk.view.tickets) if (row.group === body.text) Object.assign(row, { stage: "ended", outcome: "stopped", person: false, held: undefined });
  });
  await openTasks(page);
  await page.getByTestId("group-open").click();
  const pane = page.getByTestId("group-page");
  await pane.getByTestId("group-stop").click();
  await expect(pane.getByTestId("group-stop-confirm")).toHaveText("Stop every member");
  expect(desk.acts).toHaveLength(0);
  const sent = actSent(page);
  await pane.getByTestId("group-stop-confirm").click();
  expect((await sent).postDataJSON()).toEqual({ kind: "stopGroup", ticketId: "", triggerId: "", text: "grp-1", reason: "" });
  await expect(pane.getByTestId("group-status")).toHaveText("Split · Stopped · 0/2");
  await expect(pane.getByTestId("group-stop")).toHaveCount(0);
});

test("a comparison lines the attempts up side by side, and picking one marks it and fades the rest", async ({ page }) => {
  const desk = compareDesk();
  await mockApi(page, "C:/tmp/ws", { sessions: SESSIONS, buddies: BUDDIES });
  await mockDesk(page, desk, (body) => {
    if (body.kind === "pick") Object.assign(desk.view.groups[0], { winner: body.ticketId, verdict: "Codex's fix is smaller and keeps the session store." });
  });
  await openTasks(page);
  await page.getByRole("button", { name: /Recently ended/ }).click();
  await expect(page.getByTestId("group-progress")).toHaveText("Compare · 2/2");
  await page.getByTestId("group-open").click();
  const pane = page.getByTestId("group-page");
  await expect(pane.getByTestId("group-verdict")).toContainText("When they're all done, the router compares them in the main chat");
  const columns = pane.getByTestId("group-column");
  await expect(columns).toHaveCount(2);
  await expect(columns.nth(0)).toContainText("Codex");
  await expect(columns.nth(0)).toContainText("Fixed the token refresh");
  await expect(columns.nth(0).getByRole("button", { name: "refresh.go" })).toBeVisible({ timeout: 10_000 });
  await expect(columns.nth(0)).toContainText("1 file changed");
  await expect(columns.nth(1)).toContainText("Rewrote the session store");
  await expect(columns.getByTestId("group-pick")).toHaveCount(2);
  const sent = actSent(page);
  await columns.nth(0).getByTestId("group-pick").click();
  expect((await sent).postDataJSON()).toEqual({ kind: "pick", ticketId: "tkt-4", triggerId: "", text: "grp-2", reason: "" });
  await expect(columns.nth(0).getByTestId("group-picked")).toHaveText("Picked");
  await expect(columns.nth(1)).toHaveClass(/opacity-55/);
  await expect(columns.getByTestId("group-pick")).toHaveCount(0);
  await expect(pane.getByTestId("group-verdict")).toContainText("Codex's fix is smaller");
  await columns.nth(1).getByTestId("group-view").click();
  await expect(page.getByTestId("task-read").getByTestId("task-group-crumb")).toHaveText("Task group: Fix the login bug");
});

/** mockLive 接管会话 s1 的实时尾部。events 按下标编 seq；事件流每次只报最新的 seq，200 毫秒后重连，客户端看到新序号就回拉。 */
async function mockLive(page: Page, events: Record<string, any>[]) {
  const numbered = () => events.map((ev, i) => ({ ...ev, seq: i + 1, session_id: "s1" }));
  await page.route((url) => url.pathname === "/api/sessions/s1/live/sync", async (route) => {
    const after = Number(new URL(route.request().url()).searchParams.get("after") || 0);
    return route.fulfill({ json: { events: numbered().filter((ev) => ev.seq > after), head: events.length, gap: false, more: false } });
  });
  await page.route((url) => url.pathname === "/api/sessions/s1/events", async (route) => {
    const notice = events.length ? `data: ${JSON.stringify({ seq: events.length, session_id: "s1" })}\n\n` : "";
    return route.fulfill({ status: 200, contentType: "text/event-stream", body: `retry: 200\n\n${notice}` });
  });
}

let clock = Date.now() - 3_600_000;

/** event 是一条追踪事件，时间依次往后。 */
function event(type: string, source: string, payload: Record<string, any>): Record<string, any> {
  clock += 1000;
  return { type, source, session_id: "s1", ts: new Date(clock).toISOString(), payload };
}

/** groupDesk 是「Build the todo app」这组分工的快照：后端交给 Codex，前端交给 yoyo 并等后端。back 和 front 改写两个成员。 */
function groupDesk(state: string, back: Record<string, any>, front: Record<string, any>): Record<string, any> {
  const tickets = [
    ticket({ id: "tkt-1", title: "Write the API", role: "Backend", agent: "codex", group: "grp-1", ...back }),
    ticket({ id: "tkt-2", title: "Build the page", role: "Frontend", agent: "yoyo", group: "grp-1", dependsOn: ["tkt-1"], ...front }),
  ];
  const done = tickets.filter((row) => row.stage === "ended" && row.outcome === "completed").length;
  return deskView(tickets, [{ id: "grp-1", mode: "split", title: "Build the todo app", session: "s1", tickets: ["tkt-1", "tkt-2"], state, done, total: 2, openedAt: minutesAgo(30), updatedAt: minutesAgo(1) }]);
}

/** before 在 a 出现在 b 前面时为真。两个都按测试标记和文字找第一个。 */
async function before(page: Page, a: { testId: string; text?: string }, b: { testId: string; text?: string }): Promise<boolean> {
  return page.evaluate(([x, y]) => {
    const find = (q: { testId: string; text?: string }) => [...document.querySelectorAll(`[data-testid="${q.testId}"]`)].find((el) => !q.text || el.textContent?.includes(q.text));
    const first = find(x);
    const second = find(y);
    return !!first && !!second && !!(first.compareDocumentPosition(second) & Node.DOCUMENT_POSITION_FOLLOWING);
  }, [a, b] as const);
}

const COMPLETED = { stage: "ended", action: "", outcome: "completed", updatedAt: minutesAgo(3) };

test("two agents in one message bring a group card under the router's reply that folds member status in and collapses when done", async ({ page }) => {
  const desk: Desk = { view: deskView([], []), acts: [], transcripts: {} };
  const live: Record<string, any>[] = [];
  await mockApi(page, "C:/tmp/ws", { sessions: SESSIONS, buddies: BUDDIES });
  await mockDesk(page, desk);
  await mockLive(page, live);
  await page.goto("/");
  const box = page.getByTestId("composer-input");
  await expect(box).toHaveAttribute("contenteditable", "true", { timeout: 15_000 });
  await box.click();
  const list = page.getByRole("listbox", { name: "Mention suggestions" });
  for (const typed of ["@cod", "@yoy"]) {
    await box.pressSequentially(typed);
    await expect(list.getByText("Agents", { exact: true })).toBeVisible({ timeout: 10_000 });
    await page.keyboard.press("Enter");
  }
  await expect(page.getByTestId("mention-chip")).toHaveCount(2);
  await box.pressSequentially("build a todo app");
  const sent = page.waitForRequest((r) => r.method() === "POST" && /\/api\/sessions\/[^/]+\/messages$/.test(new URL(r.url()).pathname));
  await page.keyboard.press("Enter");
  const text = String((await sent).postDataJSON()?.text || "");
  expect(text).toContain("@agent:codex");
  expect(text).toContain("@agent:yoyo");

  desk.view = groupDesk("running", { openedAt: minutesAgo(5) }, { action: "queued", held: "deps" });
  live.push(
    event("tool_call", "agent", { id: "call-1", name: "task_split", arguments: "{\"mode\":\"split\"}" }),
    event("assistant", "worker", {
      id: "group:grp-1",
      kind: "group",
      group: "grp-1",
      title: "Build the todo app",
      mode: "split",
      members: [
        { ticket: "tkt-1", agent: "codex", agentName: "Codex", role: "Backend", title: "Write the API" },
        { ticket: "tkt-2", agent: "yoyo", role: "Frontend", title: "Build the page", dependsOn: ["tkt-1"] },
      ],
      text: "Task group “Build the todo app” · split in 2",
    }),
    event("tool_result", "agent", { id: "call-1", name: "task_split", content: "Opened task group grp-1." }),
    event("assistant", "agent", { id: "reply-1", text: "I split it in two: Codex writes the API, then yoyo builds the page." }),
    event("turn_end", "runtime", { id: "end-1" }),
  );
  const card = page.getByTestId("group-card");
  await expect(card).toHaveCount(1, { timeout: 15_000 });
  await expect(card.getByTestId("group-card-title")).toHaveText("Build the todo app");
  await expect(card).toContainText("Task group · Split");
  await expect(card.getByTestId("group-card-progress")).toHaveText("0/2", { timeout: 10_000 });
  await expect(card.locator(".progress-hairline .is-indeterminate")).toHaveCount(1);
  const members = card.getByTestId("group-card-member");
  await expect(members).toHaveCount(2);
  await expect(members.nth(0)).toContainText("Codex · Backend");
  await expect(members.nth(0).getByTestId("group-card-status")).toHaveText("Working · 5 min");
  await expect(members.nth(1)).toContainText("yoyo · Frontend");
  await expect(members.nth(1).getByTestId("group-card-status")).toHaveText("Waiting on Backend");
  expect(await before(page, { testId: "agent-turn", text: "I split it in two" }, { testId: "group-card" })).toBe(true);

  desk.view = groupDesk("attention", COMPLETED, { stage: "person", action: "answer", person: true, attention: "Should the page use dark mode?", openedAt: minutesAgo(2) });
  live.push(
    event("assistant", "worker", { id: "handoff:tkt-1:a", kind: "status", ticket: "tkt-1", state: "ready", agent: "codex", agentName: "Codex", group: "grp-1", role: "Backend", text: "The API is ready on port 8080." }),
    event("assistant", "worker", { id: "handoff:tkt-2:b", kind: "handoff", ticket: "tkt-2", state: "question", group: "grp-1", role: "Frontend", text: "Should the page use dark mode?" }),
  );
  const ask = page.getByTestId("task-handoff");
  await expect(ask).toBeVisible({ timeout: 15_000 });
  await expect(ask.getByTestId("task-event-agent")).toContainText("yoyo · Frontend asks", { timeout: 10_000 });
  await expect(ask.getByTestId("task-event-crumb")).toHaveText("Task group: Build the todo app");
  await expect(page.getByTestId("task-status")).toHaveCount(0);
  await expect(page.getByText("The API is ready on port 8080.")).toHaveCount(0);
  await expect(card.getByTestId("group-card-progress")).toHaveText("1/2", { timeout: 10_000 });
  await expect(members.nth(0).getByTestId("group-card-status")).toHaveText("Done");
  await expect(members.nth(1).getByTestId("group-card-status")).toHaveText("Asking you");
  await expect(card.getByTestId("group-card-ask")).toContainText("yoyo · Frontend asks: Should the page use dark mode?");
  await card.getByTestId("group-card-answer").click();
  await expect(page.getByTestId("task-read")).toContainText("Build the page");
  await ask.getByTestId("task-event-crumb").click();
  await expect(page.getByTestId("group-page").getByTestId("group-title")).toHaveText("Build the todo app");

  desk.view = groupDesk("done", COMPLETED, { ...COMPLETED, updatedAt: minutesAgo(1) });
  live.push(
    event("context_injection", "wake", { id: "s1:context_injection:9", name: "wake", group: "grp-1", state: "done", title: "Build the todo app", text: "This is the report of task group grp-1, not something the person said." }),
    event("assistant", "agent", { id: "reply-2", text: "Both parts are done: the API and the page are in the shared folder." }),
    event("turn_end", "runtime", { id: "end-2" }),
  );
  const receipt = page.getByTestId("wake-receipt");
  await expect(receipt).toHaveText("Task group “Build the todo app” is done", { timeout: 15_000 });
  await expect(page.getByText("This is the report of task group grp-1")).toHaveCount(0);
  await expect(card).toHaveAttribute("data-collapsed", "", { timeout: 10_000 });
  await expect(card.getByTestId("group-card-summary")).toHaveText("Task group done · 2/2");
  await expect(card.getByTestId("group-card-member")).toHaveCount(0);
  expect(await before(page, { testId: "group-card" }, { testId: "wake-receipt" })).toBe(true);
  expect(await before(page, { testId: "wake-receipt" }, { testId: "agent-turn", text: "Both parts are done" })).toBe(true);
});

test("a finished comparison collapses to the router's pick, and to the one the person picked once picked", async ({ page }) => {
  const desk = compareDesk();
  desk.view.groups[0].recommended = "tkt-4";
  const history = [
    event("user", "user", { id: "u1", text: "@agent:codex @agent:yoyo fix the login bug, each of you" }),
    event("assistant", "worker", {
      id: "group:grp-2",
      kind: "group",
      group: "grp-2",
      title: "Fix the login bug",
      mode: "compare",
      members: [{ ticket: "tkt-4", agent: "codex", agentName: "Codex" }, { ticket: "tkt-5", agent: "yoyo" }],
      text: "Task group “Fix the login bug” · compare 2",
    }),
    event("assistant", "agent", { id: "reply-1", text: "Codex and yoyo will each fix it, then I'll compare." }),
  ].map((ev, i) => ({ ...ev, seq: i + 1 }));
  await mockApi(page, "C:/tmp/ws", { sessions: SESSIONS, buddies: BUDDIES, events: history });
  await mockDesk(page, desk, (body) => {
    if (body.kind === "pick") Object.assign(desk.view.groups[0], { winner: body.ticketId });
  });
  await page.goto("/");
  const card = page.getByTestId("group-card");
  await expect(card.getByTestId("group-card-summary")).toHaveText("Task group done · 2/2 · Router recommends Codex", { timeout: 15_000 });
  await card.click();
  const columns = page.getByTestId("group-page").getByTestId("group-column");
  await expect(columns).toHaveCount(2);
  await expect(columns.nth(0).getByTestId("group-recommended")).toHaveText("Router's pick");
  await expect(columns.nth(1).getByTestId("group-recommended")).toHaveCount(0);
  await columns.nth(0).getByTestId("group-pick").click();
  await expect(columns.nth(0).getByTestId("group-picked")).toHaveText("Picked");
  await expect(columns.getByTestId("group-recommended")).toHaveCount(0);
  await expect(card.getByTestId("group-card-summary")).toHaveText("Task group done · 2/2 · Picked Codex", { timeout: 10_000 });
});

test("a group opens in the middle, and a member opened there comes back to the group", async ({ page }) => {
  await mockApi(page, "C:/tmp/ws", { sessions: SESSIONS, buddies: BUDDIES });
  await mockDesk(page, splitDesk());
  await openTasks(page);
  await page.getByTestId("group-open").click();
  await page.getByTestId("group-focus").click();
  const stage = page.getByTestId("group-stage");
  await expect(stage).toBeVisible();
  await expect(page.getByTestId("group-row")).toBeVisible();
  await stage.getByTestId("group-lane").nth(1).click();
  const task = page.getByTestId("task-stage");
  await expect(task).toBeVisible();
  await task.getByTestId("task-group-crumb").click();
  await expect(page.getByTestId("group-stage")).toBeVisible();
  await page.getByTestId("group-stage").getByRole("button", { name: "Back to the main chat" }).click();
  await expect(page.getByTestId("group-stage")).toHaveCount(0);
  await expect(page.getByTestId("composer-input")).toBeVisible();
});
