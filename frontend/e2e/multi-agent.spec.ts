/**
 * 拥有：多 agent 任务栏的端到端用例：任务组一行的展开收起和依赖提示、分工页的车道和动态、成员工单上回到任务组的面包屑、
 * 停止全部要点两次、对比页的采用，以及任务组放到中间。
 * 不拥有：任务组怎么拆、怎么放行（Go 的 group_test.go）。假后端在 mock.ts 和 buddy-mock.ts，编排台快照由这里的 mockDesk 给出。每条 test 的标题就是它断言什么。
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
