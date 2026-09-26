import { expect, test } from "@playwright/test";
import { mockApi } from "./mock";

test("opens the agent without a setup dialog", async ({ page }) => {
  await mockApi(page, "");
  await page.goto("/");
  await expect(page.getByRole("dialog", { name: /Set up this workstation/i })).toHaveCount(0);
  await expect(page.getByRole("button", { name: "New chat", exact: true })).toBeVisible();
  await expect(page.getByRole("button", { name: "Sessions", exact: true })).toBeVisible();
  await expect(page.getByRole("button", { name: "Continue last" })).toHaveCount(0);
  await expect(page.getByRole("textbox", { name: "Message" })).toBeEnabled();
});

test("new chat is available on first launch", async ({ page }) => {
  await mockApi(page, "");
  await page.goto("/");
  await expect(page.getByRole("button", { name: "New chat", exact: true })).toBeVisible();
});

test("skip link focuses the main stage", async ({ page }) => {
  await mockApi(page, "C:/tmp/ws");
  await page.goto("/");
  await expect(page.getByRole("button", { name: "New chat", exact: true })).toBeVisible();
  await page.keyboard.press("Tab");
  const skip = page.getByRole("link", { name: "Skip to conversation" });
  await expect(skip).toBeFocused();
  await skip.press("Enter");
  await expect(page.locator("#main-stage")).toBeFocused();
});

test("command palette opens", async ({ page }) => {
  await mockApi(page, "C:/tmp/ws");
  await page.goto("/");
  await expect(page.getByRole("button", { name: "New chat", exact: true })).toBeVisible();
  await page.keyboard.press("Control+k");
  await expect(page.getByPlaceholder("Jump to a thread or action…")).toBeVisible();
});

test("review sheet stays closed on a narrow viewport until opened", async ({ page }) => {
  await page.setViewportSize({ width: 900, height: 800 });
  await mockApi(page, "C:/tmp/ws");
  await page.goto("/");
  await expect(page.getByRole("dialog", { name: /Set up this workstation/i })).toHaveCount(0);
  await expect(page.getByRole("button", { name: "New chat", exact: true })).toBeVisible();
  await expect(page.getByRole("button", { name: "Toggle review" })).toHaveAttribute("aria-pressed", "false");
  await expect(page.getByRole("tablist", { name: "Review" })).toHaveCount(0);
  await page.getByRole("button", { name: "Toggle review" }).click();
  await expect(page.getByRole("tablist", { name: "Review" })).toBeVisible();
});

test("slash plan toggles the plan chip", async ({ page }) => {
  await mockApi(page, "C:/tmp/ws");
  await page.goto("/");
  const box = page.getByRole("textbox", { name: "Message" });
  await expect(box).toBeEnabled({ timeout: 15_000 });
  await box.fill("/plan");
  await box.press("Enter");
  await expect(page.getByRole("button", { name: "Mode" })).toContainText("Plan");
});

test("control shortcuts section is reachable", async ({ page }) => {
  await mockApi(page, "C:/tmp/ws");
  await page.goto("/");
  await page.getByRole("button", { name: "Settings", exact: true }).click();
  await page.getByRole("button", { name: "Shortcuts" }).click();
  await expect(page.getByRole("button", { name: "palette" })).toHaveText("Mod+K");
});

test("thread context menu can delete a chat", async ({ page }) => {
  await mockApi(page, "C:/tmp/ws", {
    sessions: [{ id: "s1", title: "Demo thread", workspace: "C:/tmp/ws" }],
  });
  await page.goto("/");
  await page.getByRole("button", { name: /Demo thread/ }).hover();
  await page.getByRole("button", { name: "Chat actions" }).click();
  await page.getByRole("menuitem", { name: "Delete" }).click();
  const dialog = page.getByRole("dialog");
  await expect(dialog).toBeVisible();
  await dialog.getByRole("button", { name: "Delete", exact: true }).click();
  await expect(page.getByRole("button", { name: /Demo thread/ })).toHaveCount(0);
});

test("clicking a session leaves skills for that chat", async ({ page }) => {
  await mockApi(page, "C:/tmp/ws", {
    sessions: [{ id: "s1", title: "Demo thread", workspace: "C:/tmp/ws" }],
    events: [
      { type: "user", session_id: "s1", ts: "2026-01-01T00:00:00Z", payload: { text: "hello from demo" } },
    ],
  });
  await page.goto("/");
  await page.getByRole("button", { name: "Skills", exact: true }).click();
  await expect(page.getByTestId("skills-workspace")).toBeVisible();
  await expect(page.getByRole("button", { name: "Back to chat" })).toHaveCount(0);
  await page.getByRole("button", { name: /Demo thread/ }).click();
  await expect(page.getByTestId("skills-workspace")).toHaveCount(0);
  await expect(page.getByTestId("conversation-column").getByText("hello from demo")).toBeVisible();
});

test("clicking a session leaves harness for that chat", async ({ page }) => {
  await mockApi(page, "C:/tmp/ws", {
    sessions: [{ id: "s1", title: "Demo thread", workspace: "C:/tmp/ws" }],
    events: [
      { type: "user", session_id: "s1", ts: "2026-01-01T00:00:00Z", payload: { text: "hello from demo" } },
    ],
  });
  await page.goto("/");
  await page.getByRole("button", { name: /^Harness/ }).click();
  await expect(page.getByTestId("harness-workspace")).toBeVisible();
  await page.getByRole("button", { name: /Demo thread/ }).click();
  await expect(page.getByTestId("harness-workspace")).toHaveCount(0);
  await expect(page.getByTestId("conversation-column").getByText("hello from demo")).toBeVisible();
});

test("module docks toggle back to chat", async ({ page }) => {
  await mockApi(page, "C:/tmp/ws", {
    sessions: [{ id: "s1", title: "Demo thread", workspace: "C:/tmp/ws" }],
  });
  await page.goto("/");
  await page.getByRole("button", { name: "Skills", exact: true }).click();
  await expect(page.getByTestId("skills-workspace")).toBeVisible();
  await page.getByRole("button", { name: "Sessions", exact: true }).click();
  await expect(page.getByTestId("skills-workspace")).toHaveCount(0);
  await expect(page.getByTestId("conversation-column")).toBeVisible();
  await page.getByRole("button", { name: "Skills", exact: true }).click();
  await expect(page.getByTestId("skills-workspace")).toBeVisible();
  await page.getByRole("button", { name: "Skills", exact: true }).click();
  await expect(page.getByTestId("skills-workspace")).toHaveCount(0);
  await expect(page.getByTestId("conversation-column")).toBeVisible();
  await page.getByRole("button", { name: /^Harness/ }).click();
  await expect(page.getByTestId("harness-workspace")).toBeVisible();
  await page.getByRole("button", { name: /^Harness/ }).click();
  await expect(page.getByTestId("harness-workspace")).toHaveCount(0);
  await expect(page.getByTestId("conversation-column")).toBeVisible();
});

test("new chat from skills opens the conversation", async ({ page }) => {
  await mockApi(page, "C:/tmp/ws", {
    sessions: [{ id: "s1", title: "Demo thread", workspace: "C:/tmp/ws" }],
  });
  await page.goto("/");
  await page.getByRole("button", { name: "Skills", exact: true }).click();
  await expect(page.getByTestId("skills-workspace")).toBeVisible();
  await page.getByRole("button", { name: "New chat", exact: true }).click();
  await expect(page.getByTestId("skills-workspace")).toHaveCount(0);
  await expect(page.getByTestId("conversation-column")).toBeVisible();
  await expect(page.getByRole("textbox", { name: "Message" })).toBeEnabled();
});

test("session hover card shows copyable session id", async ({ page }) => {
  await mockApi(page, "C:/tmp/ws", {
    sessions: [{ id: "sess-demo-id", title: "Demo thread", workspace: "C:/tmp/ws", created_at: "2026-09-20T09:00:00Z" }],
  });
  await page.goto("/");
  await page.getByRole("button", { name: /Demo thread/ }).hover();
  const card = page.getByRole("complementary", { name: "Session" });
  await expect(card).toBeVisible();
  await expect(card.getByText("sess-demo-id")).toBeVisible();
  await expect(card.getByRole("button", { name: "Copy session ID" })).toBeVisible();
  await expect(page.getByRole("menuitem", { name: "Copy session ID" })).toHaveCount(0);
});

test("provider preset fills a compatible base URL", async ({ page }) => {
  await mockApi(page, "C:/tmp/ws");
  await page.goto("/");
  await page.getByRole("button", { name: "Settings", exact: true }).click();
  await page.getByRole("button", { name: "Models", exact: true }).click();
  await page.getByRole("radio", { name: "DeepSeek" }).click();
  await expect(page.getByRole("textbox", { name: "Base URL" })).toHaveValue("https://api.deepseek.com");
  await expect(page.getByRole("textbox", { name: "Model", exact: true })).toHaveValue("deepseek-chat");
  await expect(page.getByText("DeepSeek Chat")).toBeVisible();
});

test("two-turn transcript keeps both assistant rounds", async ({ page }) => {
  await mockApi(page, "C:/tmp/ws", {
    sessions: [{ id: "s1", title: "Demo thread", workspace: "C:/tmp/ws" }],
    events: [
      { type: "user", session_id: "s1", ts: "2026-01-01T00:00:00Z", payload: { text: "first question" } },
      { type: "assistant", session_id: "s1", ts: "2026-01-01T00:00:01Z", payload: { text: "first answer", id: "s1:r1" } },
      { type: "user", session_id: "s1", ts: "2026-01-01T00:00:02Z", payload: { text: "second question" } },
      { type: "assistant", session_id: "s1", ts: "2026-01-01T00:00:03Z", payload: { text: "second answer", id: "s1:r2" } },
    ],
  });
  await page.goto("/");
  await expect(page.getByText("first question")).toBeVisible();
  await expect(page.getByText("first answer")).toBeVisible();
  await expect(page.getByText("second question")).toBeVisible();
  await expect(page.getByText("second answer")).toBeVisible();
});

test("second-turn markdown keeps headings and lists", async ({ page }) => {
  await mockApi(page, "C:/tmp/ws", {
    sessions: [{ id: "s1", title: "Demo thread", workspace: "C:/tmp/ws" }],
    events: [
      { type: "user", session_id: "s1", ts: "2026-01-01T00:00:00Z", payload: { text: "first question" } },
      { type: "assistant", session_id: "s1", ts: "2026-01-01T00:00:01Z", payload: { text: "first answer", id: "s1:r1" } },
      { type: "user", session_id: "s1", ts: "2026-01-01T00:00:02Z", payload: { text: "write a report" } },
      {
        type: "assistant",
        session_id: "s1",
        ts: "2026-01-01T00:00:03Z",
        payload: { text: "## Report\n\n- item one\n- item two\n\nDone.", id: "s1:r2" },
      },
    ],
  });
  await page.goto("/");
  const live = page.getByTestId("agent-turn").filter({ hasText: "Report" });
  await expect(live.getByRole("heading", { name: "Report" })).toBeVisible();
  await expect(live.getByText("item one")).toBeVisible();
  await expect(live.getByText("item two")).toBeVisible();
  await expect(live.getByText("Done.")).toBeVisible();
  await expect(page.getByText("first answer")).toBeVisible();
});

test("assistant letter uses a dense type ramp", async ({ page }) => {
  const body = [
    "# Findings",
    "",
    "Opening sentence that should read as the scan target.",
    "",
    "## Next step",
    "",
    "A supporting paragraph with `inline` code.",
    "",
    "- first item",
    "- second item",
    "",
    "```ts",
    "export const n = 1;",
    "```",
    "",
  ].join("\n");
  await mockApi(page, "C:/tmp/ws", {
    sessions: [{ id: "s1", title: "Demo thread", workspace: "C:/tmp/ws" }],
    events: [
      { type: "user", session_id: "s1", ts: "2026-01-01T00:00:00Z", payload: { text: "summarize the workspace" } },
      { type: "assistant", session_id: "s1", ts: "2026-01-01T00:00:01Z", payload: { text: "I'll look around.", id: "s1:r1" } },
      { type: "tool_call", session_id: "s1", ts: "2026-01-01T00:00:02Z", payload: { id: "c1", name: "read_file", arguments: JSON.stringify({ path: "README.md" }), round: "s1:r1" } },
      { type: "tool_result", session_id: "s1", ts: "2026-01-01T00:00:03Z", payload: { id: "c1", name: "read_file", content: "ok", round: "s1:r1" } },
      { type: "assistant", session_id: "s1", ts: "2026-01-01T00:00:04Z", payload: { text: body, id: "s1:r2" } },
    ],
  });
  await page.goto("/");
  const turn = page.getByTestId("agent-turn").filter({ hasText: "Findings" });
  await expect(turn.getByRole("heading", { name: "Findings" })).toBeVisible();
  const sizeOf = (locator: ReturnType<typeof turn.locator>) =>
    locator.evaluate((el) => parseFloat(getComputedStyle(el).fontSize));
  const h1 = await sizeOf(turn.getByRole("heading", { name: "Findings" }));
  const h2 = await sizeOf(turn.getByRole("heading", { name: "Next step" }));
  const prose = await sizeOf(turn.locator(".md-body p").filter({ hasText: "Opening sentence" }));
  const process = await sizeOf(turn.getByTestId("process-summary"));
  const code = await sizeOf(turn.locator('[data-streamdown="code-block"] code').first());
  expect(h1).toBeGreaterThanOrEqual(15);
  expect(h1).toBeLessThan(18);
  expect(h2).toBeGreaterThanOrEqual(13.5);
  expect(h2).toBeLessThan(h1);
  expect(prose).toBeGreaterThanOrEqual(12.5);
  expect(prose).toBeLessThanOrEqual(14.5);
  expect(process).toBeGreaterThanOrEqual(11.5);
  expect(process).toBeLessThanOrEqual(13.5);
  expect(process).toBeLessThan(prose);
  expect(code).toBeGreaterThanOrEqual(11);
  expect(code).toBeLessThanOrEqual(13);
  const headingGap = await turn.getByRole("heading", { name: "Next step" }).evaluate((el) => parseFloat(getComputedStyle(el).marginTop));
  expect(headingGap).toBeLessThan(20);
});

test("copy sits under the message, not the column corner", async ({ page }) => {
  await mockApi(page, "C:/tmp/ws", {
    sessions: [{ id: "s1", title: "Demo thread", workspace: "C:/tmp/ws" }],
    events: [
      { type: "user", session_id: "s1", ts: "2026-01-01T00:00:00Z", payload: { text: "first question" } },
      { type: "assistant", session_id: "s1", ts: "2026-01-01T00:00:01Z", payload: { text: "first answer", id: "s1:r1" } },
    ],
  });
  await page.goto("/");
  const answer = page.getByText("first answer");
  await expect(answer).toBeVisible();
  const turn = page.getByTestId("agent-turn").filter({ hasText: "first answer" });
  await turn.hover();
  const copyBtn = turn.getByRole("button", { name: "Copy" });
  await expect(copyBtn).toBeVisible();
  const a = await answer.boundingBox();
  const c = await copyBtn.boundingBox();
  expect(a && c).toBeTruthy();
  expect(c!.y).toBeGreaterThan(a!.y);
  expect(c!.x).toBeLessThan(a!.x + a!.width + 48);
});

test("one copy control per agent turn", async ({ page }) => {
  await mockApi(page, "C:/tmp/ws", {
    sessions: [{ id: "s1", title: "Demo thread", workspace: "C:/tmp/ws" }],
    events: [
      { type: "user", session_id: "s1", ts: "2026-01-01T00:00:00Z", payload: { text: "look around" } },
      { type: "assistant", session_id: "s1", ts: "2026-01-01T00:00:01Z", payload: { text: "looking around", id: "s1:r1" } },
      { type: "tool_call", session_id: "s1", ts: "2026-01-01T00:00:02Z", payload: { id: "c1", name: "read_file", arguments: "{}", round: "s1:r1" } },
      { type: "tool_result", session_id: "s1", ts: "2026-01-01T00:00:03Z", payload: { id: "c1", name: "read_file", content: "ok", round: "s1:r1" } },
      { type: "assistant", session_id: "s1", ts: "2026-01-01T00:00:04Z", payload: { text: "found it", id: "s1:r2" } },
      { type: "user", session_id: "s1", ts: "2026-01-01T00:00:05Z", payload: { text: "thanks" } },
      { type: "assistant", session_id: "s1", ts: "2026-01-01T00:00:06Z", payload: { text: "anytime", id: "s1:r3" } },
    ],
  });
  await page.goto("/");
  await expect(page.getByText("looking around")).toBeVisible();
  await expect(page.getByText("found it")).toBeVisible();
  const turns = page.getByTestId("agent-turn");
  await expect(turns).toHaveCount(2);
  await expect(turns.nth(0).getByRole("button", { name: "Copy" })).toHaveCount(1);
  await expect(turns.nth(0).getByRole("button", { name: "Copy" })).toHaveAttribute("data-copy-text", "looking around\n\nfound it");
  await expect(turns.nth(1).getByRole("button", { name: "Copy" })).toHaveCount(1);
  await expect(turns.nth(1).getByRole("button", { name: "Copy" })).toHaveAttribute("data-copy-text", "anytime");
});

test("in-progress caret stays on the live assistant", async ({ page }) => {
  await mockApi(page, "C:/tmp/ws", {
    sessions: [{ id: "s1", title: "Demo thread", workspace: "C:/tmp/ws" }],
    running: true,
    events: [
      { type: "user", session_id: "s1", ts: "2026-01-01T00:00:00Z", payload: { text: "first question" } },
      { type: "assistant", session_id: "s1", ts: "2026-01-01T00:00:01Z", payload: { text: "first answer", id: "s1:r1" } },
      { type: "user", session_id: "s1", ts: "2026-01-01T00:00:02Z", payload: { text: "second question" } },
      { type: "assistant", session_id: "s1", ts: "2026-01-01T00:00:03Z", payload: { text: "second answer", id: "s1:r2" } },
      { type: "tool_call", session_id: "s1", ts: "2026-01-01T00:00:04Z", payload: { id: "c1", name: "read_file", arguments: "{}" } },
    ],
  });
  await page.goto("/");
  await expect(page.getByText("first answer")).toBeVisible();
  await expect(page.locator(".assistant-live")).toHaveCount(0);
  await expect(page.getByTestId("process-group")).toHaveAttribute("data-live", "true");
  await expect(page.getByTestId("process-summary")).toHaveCount(0);
  await expect(page.getByTestId("tool-row").filter({ hasText: "read_file" })).toBeVisible();
});

test("streaming caret sits on the open assistant", async ({ page }) => {
  await mockApi(page, "C:/tmp/ws", {
    sessions: [{ id: "s1", title: "Demo thread", workspace: "C:/tmp/ws" }],
    running: true,
    events: [
      { type: "user", session_id: "s1", ts: "2026-01-01T00:00:00Z", payload: { text: "first question" } },
      { type: "assistant", session_id: "s1", ts: "2026-01-01T00:00:01Z", payload: { text: "old answer", id: "s1:r1" } },
      { type: "user", session_id: "s1", ts: "2026-01-01T00:00:02Z", payload: { text: "second question" } },
      { type: "assistant", session_id: "s1", ts: "2026-01-01T00:00:03Z", payload: { text: "", id: "s1:r2", delta: true } },
      { type: "assistant", session_id: "s1", ts: "2026-01-01T00:00:04Z", payload: { text: "live answer", id: "s1:r2", delta: true } },
    ],
  });
  await page.goto("/");
  await expect(page.getByText("live answer")).toBeVisible();
  const live = page.getByTestId("agent-turn").filter({ hasText: "live answer" });
  const old = page.getByTestId("agent-turn").filter({ hasText: "old answer" });
  await expect(live.locator(".assistant-live")).toHaveCount(1);
  await expect(old.locator(".assistant-live")).toHaveCount(0);
});

test("a finished round keeps one letter and thinks at the foot", async ({ page }) => {
  await mockApi(page, "C:/tmp/ws", {
    sessions: [{ id: "s1", title: "Demo thread", workspace: "C:/tmp/ws" }],
    running: true,
    events: [
      { type: "user", session_id: "s1", ts: "2026-01-01T00:00:00Z", payload: { text: "look around" } },
      { type: "assistant", session_id: "s1", ts: "2026-01-01T00:00:01Z", payload: { text: "looking around", id: "s1:r1" } },
    ],
  });
  await page.goto("/");
  const turn = page.getByTestId("agent-turn");
  await expect(turn.getByText("looking around")).toBeVisible();
  await expect(turn.locator(".assistant-live")).toHaveCount(0);
  await expect(turn.getByTestId("working-line")).toBeVisible();
  await expect(page.getByTestId("working-line")).toHaveCount(1);
});

test("continue this turn posts retry instead of a new user message", async ({ page }) => {
  await mockApi(page, "C:/tmp/ws", {
    sessions: [{ id: "s1", title: "Demo thread", workspace: "C:/tmp/ws" }],
    events: [
      { type: "user", session_id: "s1", ts: "2026-01-01T00:00:00Z", payload: { text: "first question" } },
      { type: "assistant", session_id: "s1", ts: "2026-01-01T00:00:01Z", payload: { text: "first answer", id: "s1:r1" } },
      { type: "error", session_id: "s1", ts: "2026-01-01T00:00:02Z", payload: { kind: "provider", title: "The model endpoint failed", retryable: true, error: "boom" } },
    ],
  });
  await page.goto("/");
  const retry = page.getByRole("button", { name: "Continue this turn" });
  await expect(retry).toBeVisible();
  const posted = page.waitForRequest((req) => req.method() === "POST" && /\/api\/sessions\/[^/]+\/retry$/.test(new URL(req.url()).pathname));
  const leaked = page.waitForRequest((req) => req.method() === "POST" && req.url().includes("/messages"), { timeout: 1500 }).then(() => true).catch(() => false);
  await retry.click();
  await posted;
  expect(await leaked).toBe(false);
});

test("review pane has files browser and trace", async ({ page }) => {
  await mockApi(page, "C:/tmp/ws");
  await page.goto("/");
  await page.getByRole("button", { name: "Toggle review" }).click();
  await expect(page.getByRole("tablist", { name: "Review" })).toBeVisible();
  await expect(page.getByRole("tab", { name: /^Files/ })).toBeVisible();
  await expect(page.getByRole("tab", { name: /^Browser/ })).toBeVisible();
  await expect(page.getByRole("tab", { name: /^Trace/ })).toBeVisible();
  await expect(page.getByRole("tab", { name: /^Changes/ })).toHaveCount(0);
  await expect(page.getByRole("tab", { name: /^Diff/ })).toHaveCount(0);
  await expect(page.getByRole("tab", { name: /^Queue/ })).toHaveCount(0);
  await expect(page.getByRole("tab", { name: /^Memory/ })).toHaveCount(0);
  await expect(page.getByRole("tab", { name: /Context/ })).toHaveCount(0);
  await expect(page.getByRole("tab", { name: /Ask/ })).toHaveCount(0);
  await expect(page.getByTestId("review-file-tree")).toBeVisible();
  await expect(page.getByTestId("file-tree-src/main.go")).toBeVisible();
  await page.getByRole("button", { name: "Inbox" }).click();
  await expect(page.getByTestId("inbox-menu")).toBeVisible();
  await expect(page.getByTestId("inbox-menu")).toContainText("Nothing is waiting on you.");
});

test("trace tab shows trajectory and artifacts", async ({ page }) => {
  await mockApi(page, "C:/tmp/ws", {
    sessions: [{ id: "s1", title: "Demo thread", workspace: "C:/tmp/ws" }],
    events: [
      { type: "user", session_id: "s1", ts: "2026-01-01T00:00:00Z", payload: { text: "list files" } },
      { type: "tool_call", session_id: "s1", ts: "2026-01-01T00:00:01Z", payload: { name: "shell", arguments: "{\"cmd\":\"ls\"}", id: "c1" } },
      { type: "tool_result", session_id: "s1", ts: "2026-01-01T00:00:02Z", payload: { name: "shell", content: "a.txt", id: "c1", spill_id: "c1", bytes: 5, elapsed_ms: 12 } },
    ],
    artifacts: [{ kind: "spill", id: "c1", label: "c1", bytes: 5 }],
    spill: { c1: { id: "c1", bytes: 5, text: "a.txt", truncated: false } },
  });
  await page.goto("/");
  await page.getByRole("button", { name: "Toggle review" }).click();
  await page.getByRole("tab", { name: /^Trace/ }).click();
  await expect(page.getByTestId("trace-panel")).toBeVisible();
  await expect(page.getByTestId("trace-event").filter({ hasText: "list files" })).toBeVisible();
  await expect(page.getByTestId("trace-event").filter({ hasText: "shell" }).first()).toBeVisible();
  await expect(page.getByTestId("trace-panel").getByText("c1").first()).toBeVisible();
});

test("composer exposes steer while a turn is running", async ({ page }) => {
  await mockApi(page, "C:/tmp/ws", {
    sessions: [{ id: "s1", title: "Demo thread", workspace: "C:/tmp/ws" }],
    running: true,
    events: [{ type: "user", session_id: "s1", ts: "2026-01-01T00:00:00Z", payload: { text: "go" } }],
  });
  await page.goto("/");
  await expect(page.getByRole("button", { name: "Steer" })).toBeVisible();
  await expect(page.getByRole("button", { name: "Stop" })).toBeVisible();
});

test("model picker stays on the configured provider", async ({ page }) => {
  await mockApi(page, "C:/tmp/ws", {
    sessions: [{ id: "s1", title: "Demo thread", workspace: "C:/tmp/ws", model: "deepseek-chat" }],
    config: { provider: "deepseek", model: "deepseek-chat", models: ["deepseek-chat", "deepseek-reasoner"] },
  });
  await page.goto("/");
  await page.getByRole("combobox", { name: "Model" }).click();
  await expect(page.getByRole("option", { name: "DeepSeek Chat" })).toBeVisible();
  await expect(page.getByRole("option", { name: "DeepSeek Reasoner" })).toBeVisible();
  await expect(page.getByRole("option", { name: /gpt-4/i })).toHaveCount(0);
  await expect(page.getByRole("option", { name: /claude/i })).toHaveCount(0);
});

test("composer shows context breakdown", async ({ page }) => {
  await mockApi(page, "C:/tmp/ws", {
    sessions: [{ id: "s1", title: "Demo thread", workspace: "C:/tmp/ws" }],
    context: {
      tokens: 18000,
      budget: 100000,
      window: 128000,
      prefix_tokens: 2200,
      schema_tokens: 4100,
      dynamic_tokens: 800,
      layers: ["budget"],
      elided: 2,
    },
  });
  await page.goto("/");
  const meter = page.getByRole("group", { name: "Context" }).first();
  await expect(meter).toBeVisible();
  await expect(meter.getByText("System", { exact: true })).toBeVisible();
  await expect(meter.getByText("Tools", { exact: true })).toBeVisible();
  await expect(meter.getByText("Working", { exact: true })).toBeVisible();
  await expect(meter.getByText("Chat", { exact: true })).toBeVisible();
  await expect(meter.getByText("Free", { exact: true })).toBeVisible();
});

test("approval is asked in the transcript", async ({ page }) => {
  await mockApi(page, "C:/tmp/ws", {
    sessions: [{ id: "s1", title: "Demo thread", workspace: "C:/tmp/ws" }],
    events: [
      { type: "user", session_id: "s1", ts: "2026-01-01T00:00:00Z", payload: { text: "patch readme" } },
    ],
    approvals: [{ id: "a1", request: { action: "write_file", command: "write README.md", path: "README.md", level: "write" } }],
  });
  await page.goto("/");
  await expect(page.getByText("Needs approval")).toBeVisible();
  await expect(page.getByRole("button", { name: "Allow once", exact: true })).toBeVisible();
  await expect(page.getByRole("button", { name: "Reject" })).toBeVisible();
  await expect(page.getByRole("tab", { name: /Ask/ })).toHaveCount(0);
});

test("conversation and composer share a fluid stage column", async ({ page }) => {
  await mockApi(page, "C:/tmp/ws", {
    sessions: [{ id: "s1", title: "Demo thread", workspace: "C:/tmp/ws" }],
    events: [
      { type: "user", session_id: "s1", ts: "2026-01-01T00:00:00Z", payload: { text: "first question" } },
      { type: "assistant", session_id: "s1", ts: "2026-01-01T00:00:01Z", payload: { text: "first answer", id: "s1:r1" } },
    ],
  });
  await page.goto("/");
  const toggle = page.getByRole("button", { name: "Toggle review" });
  await expect(toggle).toBeVisible();
  if ((await toggle.getAttribute("aria-pressed")) === "true") {
    await toggle.click();
  }
  const thread = page.getByTestId("conversation-column");
  const composer = page.getByTestId("composer-column");
  const turn = page.getByTestId("agent-turn");
  await expect(thread).toBeVisible();
  await expect(composer).toBeVisible();
  await expect(turn).toBeVisible();

  await page.setViewportSize({ width: 1200, height: 900 });
  const narrowPane = await thread.boundingBox();
  const narrowComposer = await composer.boundingBox();
  const narrowTurn = await turn.boundingBox();
  expect(narrowPane && narrowComposer && narrowTurn).toBeTruthy();
  expect(narrowComposer!.width).toBeGreaterThan(narrowPane!.width * 0.8);
  expect(narrowComposer!.width).toBeLessThanOrEqual(narrowPane!.width + 1);
  expect(narrowTurn!.width).toBeGreaterThan(narrowComposer!.width * 0.9);

  await page.setViewportSize({ width: 1680, height: 900 });
  const wideComposer = await composer.boundingBox();
  const wideTurn = await turn.boundingBox();
  expect(wideComposer && wideTurn).toBeTruthy();
  expect(wideComposer!.width).toBeGreaterThan(narrowComposer!.width + 40);
  expect(wideTurn!.width).toBeGreaterThan(narrowTurn!.width + 40);
});

test("insert mention keeps the popup open and pins a file token", async ({ page }) => {
  await mockApi(page, "C:/tmp/ws", {
    sessions: [{ id: "s1", title: "Demo thread", workspace: "C:/tmp/ws" }],
  });
  await page.goto("/");
  const box = page.getByRole("textbox", { name: "Message" });
  await expect(box).toBeEnabled({ timeout: 15_000 });
  await box.click();
  await box.pressSequentially("@");
  await expect(page.getByRole("listbox", { name: "Mentions" })).toBeVisible();
  await page.getByRole("option", { name: /@file:/ }).first().click();
  await expect(page.getByRole("option", { name: /@file:src\/main.go/ })).toBeVisible();
  await page.getByRole("option", { name: /@file:src\/main.go/ }).click();
  await expect(page.getByTestId("mention-chip")).toHaveAttribute("aria-label", "@file:src/main.go");
  await expect(page.getByTestId("mention-chip")).toContainText("main.go");
  await expect(page.getByRole("button", { name: "More" })).toHaveCount(0);
});

test("installed skills appear in @skill and keyboard highlight stays in view", async ({ page }) => {
  const skills = [
    ...Array.from({ length: 14 }, (_, i) => ({
      name: `bundled-${String(i).padStart(2, "0")}`,
      description: "bundled pack",
      source: "bundled",
    })),
    { name: "arxiv-watcher", description: "watch papers", source: "market", display_name: "Arxiv Watcher" },
    ...Array.from({ length: 18 }, (_, i) => ({
      name: `pack-${String(i).padStart(2, "0")}`,
      description: `installed ${i}`,
      source: "market",
    })),
  ];
  await mockApi(page, "C:/tmp/ws", {
    sessions: [{ id: "s1", title: "Demo thread", workspace: "C:/tmp/ws" }],
    skills,
  });
  await page.goto("/");
  const box = page.getByRole("textbox", { name: "Message" });
  await expect(box).toBeEnabled({ timeout: 15_000 });
  await box.click();
  await box.pressSequentially("@");
  await page.getByRole("option", { name: /@skill:/ }).click();
  await expect(page.getByRole("option", { name: /@skill:arxiv-watcher/ })).toBeVisible();
  await box.pressSequentially("pack-17");
  const list = page.getByRole("listbox", { name: "Mentions" });
  await expect(list.getByRole("option", { name: /@skill:pack-17/ })).toBeVisible();
  await box.fill("");
  await box.pressSequentially("@skill:");
  for (let i = 0; i < 16; i++) await box.press("ArrowDown");
  const selected = list.getByRole("option", { selected: true });
  await expect(selected).toBeVisible();
  const listBox = await list.boundingBox();
  const selBox = await selected.boundingBox();
  expect(listBox && selBox).toBeTruthy();
  expect(selBox!.y).toBeGreaterThanOrEqual(listBox!.y - 2);
  expect(selBox!.y + selBox!.height).toBeLessThanOrEqual(listBox!.y + listBox!.height + 2);
});

test("installed skills do not need pinning to a chat", async ({ page }) => {
  await mockApi(page, "C:/tmp/ws", {
    sessions: [{ id: "s1", title: "Demo thread", workspace: "C:/tmp/ws" }],
    skills: [{ name: "arxiv-watcher", description: "watch papers", source: "market" }],
  });
  await page.goto("/");
  await page.getByRole("button", { name: "Skills", exact: true }).click();
  await expect(page.getByTestId("skills-workspace")).toBeVisible();
  await page.getByRole("tab", { name: "Installed" }).click();
  await expect(page.getByText("arxiv-watcher")).toBeVisible();
  await expect(page.getByRole("button", { name: "Pin to chat" })).toHaveCount(0);
  await expect(page.getByRole("button", { name: "钉到对话" })).toHaveCount(0);
});

test("session authorization mode is on the composer", async ({ page }) => {
  await mockApi(page, "C:/tmp/ws", {
    sessions: [{ id: "s1", title: "Demo thread", workspace: "C:/tmp/ws", auth_mode: "default" }],
  });
  await page.goto("/");
  const auth = page.getByRole("combobox", { name: "Authorization" });
  await expect(auth).toBeVisible();
  await auth.click();
  await page.getByRole("option", { name: "Full access" }).click();
  await expect(auth).toHaveText("Full access");
});

test("composer mode menu switches agent and plan", async ({ page }) => {
  await mockApi(page, "C:/tmp/ws", {
    sessions: [{ id: "s1", title: "Demo thread", workspace: "C:/tmp/ws" }],
  });
  await page.goto("/");
  const mode = page.getByRole("button", { name: "Mode" });
  await expect(mode).toContainText("Agent");
  await mode.click();
  await page.getByRole("menuitem", { name: /Plan/ }).click();
  await expect(mode).toContainText("Plan");
});

test("session workspace is on the composer", async ({ page }) => {
  await mockApi(page, "C:/tmp/ws", {
    sessions: [
      { id: "s1", title: "Demo thread", workspace: "C:/tmp/ws" },
      { id: "s2", title: "Other", workspace: "C:/tmp/other" },
    ],
  });
  await page.goto("/");
  const picker = page.getByTestId("composer-column").getByRole("button", { name: "Workspace" });
  await expect(picker).toBeVisible();
  await expect(picker).toContainText("ws");
  await expect(page.getByTitle("C:/tmp/ws")).toHaveCount(1);
  await picker.click();
  await page.getByRole("menuitem", { name: /other/ }).click();
  await expect(picker).toContainText("other");
});

test("settled tools collapse until the operator expands them", async ({ page }) => {
  await mockApi(page, "C:/tmp/ws", {
    sessions: [{ id: "s1", title: "Demo thread", workspace: "C:/tmp/ws" }],
    events: [
      { type: "user", session_id: "s1", ts: "2026-01-01T00:00:00Z", payload: { text: "read the file" } },
      { type: "tool_call", session_id: "s1", ts: "2026-01-01T00:00:01Z", payload: { id: "c1", name: "read_file", arguments: "{\"path\":\"src/main.go\"}" } },
      { type: "tool_result", session_id: "s1", ts: "2026-01-01T00:00:02Z", payload: { id: "c1", name: "read_file", content: "package main", elapsed_ms: 12 } },
      { type: "assistant", session_id: "s1", ts: "2026-01-01T00:00:03Z", payload: { text: "it is a go file", id: "s1:r1" } },
    ],
  });
  await page.goto("/");
  await expect(page.getByText("it is a go file")).toBeVisible();
  const summary = page.getByTestId("process-summary");
  await expect(summary).toBeVisible();
  await expect(summary).toContainText("Worked for 1s");
  await expect(summary).toContainText("Read 1 file");
  await expect(page.getByText("src/main.go")).toHaveCount(0);
  await expect(page.getByText("{\"path\":\"src/main.go\"}")).toHaveCount(0);
  await summary.click();
  const row = page.getByTestId("tool-row").filter({ hasText: "read_file" });
  await expect(row).toBeVisible();
  await expect(row.getByText("src/main.go")).toBeVisible();
  await expect(row.getByText("12ms")).toBeVisible();
  await expect(page.getByText("{\"path\":\"src/main.go\"}")).toHaveCount(0);
});

test("a call without a result on a finished turn reads interrupted, never running", async ({ page }) => {
  await mockApi(page, "C:/tmp/ws", {
    sessions: [{ id: "s1", title: "Demo thread", workspace: "C:/tmp/ws" }],
    running: false,
    events: [
      { type: "user", session_id: "s1", ts: "2026-01-01T00:00:00Z", payload: { text: "read the file" } },
      { type: "tool_call", session_id: "s1", ts: "2026-01-01T00:00:01Z", payload: { id: "c1", name: "read_file", arguments: "{\"path\":\"src/main.go\"}" } },
      { type: "tool_result", session_id: "s1", ts: "2026-01-01T00:00:02Z", payload: { id: "c1", name: "read_file", content: "package main", elapsed_ms: 12 } },
      { type: "tool_call", session_id: "s1", ts: "2026-01-01T00:00:03Z", payload: { id: "c2", name: "bash", arguments: "{\"cmd\":\"go test ./...\"}" } },
    ],
  });
  await page.goto("/");
  const group = page.getByTestId("process-group");
  await expect(group).toHaveAttribute("data-live", "false");
  await expect(page.getByTestId("working-line")).toHaveCount(0);
  const summary = page.getByTestId("process-summary");
  await expect(summary).toContainText("Interrupted");
  await expect(summary).not.toContainText("Running");
  await summary.click();
  const row = page.getByTestId("tool-row").filter({ hasText: "bash" });
  await expect(row).toBeVisible();
  await expect(row).toContainText("Interrupted");
  await expect(row.locator(".pulse-dot")).toHaveCount(0);
  await expect(page.getByTestId("tool-row").filter({ hasText: "read_file" })).not.toContainText("Interrupted");
});

test("office artifacts land as review cards", async ({ page }) => {
  await mockApi(page, "C:/tmp/ws", {
    sessions: [{ id: "s1", title: "Demo thread", workspace: "C:/tmp/ws" }],
    events: [
      { type: "user", session_id: "s1", ts: "2026-01-01T00:00:00Z", payload: { text: "write the weekly" } },
      { type: "tool_call", session_id: "s1", ts: "2026-01-01T00:00:01Z", payload: { id: "c1", name: "office_create", arguments: "{\"path\":\"reports/week.docx\",\"kind\":\"docx\"}" } },
      { type: "tool_result", session_id: "s1", ts: "2026-01-01T00:00:02Z", payload: { id: "c1", name: "office_create", content: "wrote reports/week.docx", path: "reports/week.docx" } },
      { type: "assistant", session_id: "s1", ts: "2026-01-01T00:00:03Z", payload: { text: "weekly is ready", id: "s1:r1" } },
    ],
  });
  await page.goto("/");
  const card = page.getByTestId("artifact-card");
  await expect(card).toBeVisible();
  await expect(card.getByText("week.docx", { exact: true })).toBeVisible();
  await expect(card.getByText("reports/week.docx")).toBeVisible();
  await expect(card.getByRole("button", { name: "Open in Review" })).toBeVisible();
  await expect(card.getByRole("button", { name: "Open", exact: true })).toHaveCount(0);
  await card.getByRole("button", { name: "Open in Review" }).click();
  await expect(page.getByRole("tab", { name: /^Files/ })).toHaveAttribute("aria-selected", "true");
});

test("review file preview fills the pane and highlights source", async ({ page }) => {
  await mockApi(page, "C:/tmp/ws", {
    sessions: [{ id: "s1", title: "Demo thread", workspace: "C:/tmp/ws" }],
    events: [
      { type: "user", session_id: "s1", ts: "2026-01-01T00:00:00Z", payload: { text: "fix the name" } },
      {
        type: "tool_call",
        session_id: "s1",
        ts: "2026-01-01T00:00:01Z",
        payload: {
          id: "e1",
          name: "str_replace",
          arguments: JSON.stringify({ path: "src/main.go", old_str: "oldFn", new_str: "newFn" }),
        },
      },
      { type: "tool_result", session_id: "s1", ts: "2026-01-01T00:00:02Z", payload: { id: "e1", name: "str_replace", content: "replaced 1 occurrence(s) in src/main.go" } },
      { type: "assistant", session_id: "s1", ts: "2026-01-01T00:00:03Z", payload: { text: "renamed", id: "s1:r1" } },
    ],
  });
  await page.goto("/");
  const card = page.getByTestId("artifact-card").filter({ hasText: "main.go" });
  await expect(card).toBeVisible();
  await card.getByRole("button", { name: "Open in Review" }).click();
  const pane = page.getByTestId("review-files");
  const preview = page.getByTestId("review-file-preview");
  await expect(preview).toBeVisible();
  const paneBox = await pane.boundingBox();
  const prevBox = await preview.boundingBox();
  expect(paneBox && prevBox).toBeTruthy();
  expect(prevBox!.height).toBeGreaterThan(paneBox!.height * 0.55);
  const code = preview.getByTestId("artifact-code");
  await expect(code).toContainText("package main");
  await expect(code).toContainText("func Hello()");
  await expect(code.locator("span[style*='color']").first()).toBeVisible({ timeout: 15_000 });
});

test("write_file html and str_replace render as previewable artifacts", async ({ page }) => {
  await mockApi(page, "C:/tmp/ws", {
    sessions: [{ id: "s1", title: "Demo thread", workspace: "C:/tmp/ws" }],
    events: [
      { type: "user", session_id: "s1", ts: "2026-01-01T00:00:00Z", payload: { text: "build the page" } },
      {
        type: "tool_call",
        session_id: "s1",
        ts: "2026-01-01T00:00:01Z",
        payload: {
          id: "w1",
          name: "write_file",
          arguments: JSON.stringify({
            path: "web/index.html",
            content: "<!doctype html><html><body><h1>Hello RBAC</h1></body></html>",
          }),
        },
      },
      { type: "tool_result", session_id: "s1", ts: "2026-01-01T00:00:02Z", payload: { id: "w1", name: "write_file", content: "wrote web/index.html" } },
      {
        type: "tool_call",
        session_id: "s1",
        ts: "2026-01-01T00:00:03Z",
        payload: {
          id: "e1",
          name: "str_replace",
          arguments: JSON.stringify({ path: "src/main.go", old_str: "oldFn", new_str: "newFn" }),
        },
      },
      { type: "tool_result", session_id: "s1", ts: "2026-01-01T00:00:04Z", payload: { id: "e1", name: "str_replace", content: "replaced 1 occurrence(s) in src/main.go" } },
    ],
  });
  await page.goto("/");
  await expect(page.getByTestId("artifact-card")).toHaveCount(2);
  await expect(page.getByTestId("html-preview")).toBeVisible();
  const diff = page.getByTestId("artifact-diff");
  await expect(diff).toBeVisible();
  await expect(diff).toContainText("oldFn");
  await expect(diff).toContainText("newFn");
  await expect(page.getByTestId("artifact-card").first()).toContainText("index.html");
});

test("html artifacts open in the review browser pane", async ({ page }) => {
  await mockApi(page, "C:/tmp/ws", {
    sessions: [{ id: "s1", title: "Demo thread", workspace: "C:/tmp/ws" }],
    events: [
      { type: "user", session_id: "s1", ts: "2026-01-01T00:00:00Z", payload: { text: "build the page" } },
      {
        type: "tool_call",
        session_id: "s1",
        ts: "2026-01-01T00:00:01Z",
        payload: {
          id: "w1",
          name: "write_file",
          arguments: JSON.stringify({
            path: "web/index.html",
            content: "<!doctype html><html><body><h1>Hello RBAC</h1></body></html>",
          }),
        },
      },
      { type: "tool_result", session_id: "s1", ts: "2026-01-01T00:00:02Z", payload: { id: "w1", name: "write_file", content: "wrote web/index.html" } },
    ],
  });
  await page.goto("/");
  await page.getByTestId("artifact-card").getByRole("button", { name: "Open in Review" }).click();
  await expect(page.getByRole("tab", { name: "Browser" })).toHaveAttribute("aria-selected", "true");
  await expect(page.getByTestId("review-browser")).toBeVisible();
  await expect(page.getByTestId("review-browser").getByTestId("html-preview")).toBeVisible();
});

test("review browser follows a live isolated page", async ({ page }) => {
  const gif = "data:image/gif;base64,R0lGODlhAQABAIAAAAAAAP///ywAAAAAAQABAAACAUwAOw==";
  await mockApi(page, "C:/tmp/ws", {
    sessions: [{ id: "s1", title: "Demo thread", workspace: "C:/tmp/ws" }],
    events: [{ type: "user", session_id: "s1", ts: "2026-01-01T00:00:00Z", payload: { text: "open it" } }],
    browser: {
      live: true,
      url: "https://example.com",
      title: "Example",
      lane: "isolated",
      headed: false,
      screenshot: gif,
      text: "Example Domain",
      log: [{ op: "open", detail: "https://example.com" }],
    },
  });
  await page.goto("/");
  await page.getByRole("button", { name: "Toggle review" }).click();
  await page.getByRole("tab", { name: "Browser" }).click();
  await expect(page.getByTestId("review-browser")).toBeVisible();
  await expect(page.getByTestId("browser-frame")).toBeVisible();
  await expect(page.getByTestId("browser-url")).toContainText("example.com");
  await expect(page.getByTestId("browser-takeover")).toBeVisible();
  await expect(page.getByText("open", { exact: true })).toBeVisible();
});

test("write_file of source stays in the process rail without a preview card", async ({ page }) => {
  await mockApi(page, "C:/tmp/ws", {
    sessions: [{ id: "s1", title: "Demo thread", workspace: "C:/tmp/ws" }],
    events: [
      { type: "user", session_id: "s1", ts: "2026-01-01T00:00:00Z", payload: { text: "add main" } },
      {
        type: "tool_call",
        session_id: "s1",
        ts: "2026-01-01T00:00:01Z",
        payload: {
          id: "w1",
          name: "write_file",
          arguments: JSON.stringify({
            path: "src/main.go",
            content: "package main\n\nfunc main() {\n\tprintln(\"ok\")\n}\n",
          }),
        },
      },
      { type: "tool_result", session_id: "s1", ts: "2026-01-01T00:00:02Z", payload: { id: "w1", name: "write_file", content: "wrote src/main.go" } },
      {
        type: "tool_call",
        session_id: "s1",
        ts: "2026-01-01T00:00:03Z",
        payload: {
          id: "e1",
          name: "str_replace",
          arguments: JSON.stringify({ path: "src/main.go", old_str: "println", new_str: "fmt.Println" }),
        },
      },
      { type: "tool_result", session_id: "s1", ts: "2026-01-01T00:00:04Z", payload: { id: "e1", name: "str_replace", content: "replaced 1 occurrence(s) in src/main.go" } },
    ],
  });
  await page.goto("/");
  await expect(page.getByTestId("artifact-card")).toHaveCount(1);
  await expect(page.getByTestId("artifact-card")).toContainText("main.go");
  await expect(page.getByTestId("artifact-diff")).toBeVisible();
  await expect(page.getByTestId("artifact-diff")).toContainText("println");
  await expect(page.getByTestId("artifact-code")).toHaveCount(0);
  const summary = page.getByTestId("process-summary");
  await expect(summary).toBeVisible();
  await summary.click();
  await expect(page.getByTestId("tool-row").filter({ hasText: "write_file" })).toBeVisible();
});

test("assistant fenced code stays inside the letter", async ({ page }) => {
  await mockApi(page, "C:/tmp/ws", {
    sessions: [{ id: "s1", title: "Demo thread", workspace: "C:/tmp/ws" }],
    events: [
      { type: "user", session_id: "s1", ts: "2026-01-01T00:00:00Z", payload: { text: "show the snippet" } },
      {
        type: "assistant",
        session_id: "s1",
        ts: "2026-01-01T00:00:01Z",
        payload: {
          id: "s1:r1",
          text: "Here:\n\n```go\npackage main\n\nfunc Hello() string {\n\treturn \"yoyo\"\n}\n```\n",
        },
      },
    ],
  });
  await page.goto("/");
  const letter = page.getByTestId("agent-turn");
  const block = letter.locator('[data-streamdown="code-block"]');
  await expect(block).toBeVisible();
  await expect(block).toContainText("func Hello()");
  const blockBox = await block.boundingBox();
  const letterBox = await letter.boundingBox();
  expect(blockBox && letterBox).toBeTruthy();
  expect(blockBox!.width).toBeLessThanOrEqual(letterBox!.width + 1);
  expect(blockBox!.height).toBeGreaterThan(40);
  expect(blockBox!.height).toBeLessThan(280);
});

test("task plan sits above the composer instead of the letter", async ({ page }) => {
  await mockApi(page, "C:/tmp/ws", {
    sessions: [{ id: "s1", title: "Demo thread", workspace: "C:/tmp/ws" }],
    events: [
      { type: "user", session_id: "s1", ts: "2026-01-01T00:00:00Z", payload: { text: "do the work" } },
      { type: "assistant", session_id: "s1", ts: "2026-01-01T00:00:01Z", payload: { text: "on it", id: "s1:r1" } },
      {
        type: "tool_call",
        session_id: "s1",
        ts: "2026-01-01T00:00:02Z",
        payload: {
          id: "p1",
          name: "update_plan",
          arguments: JSON.stringify({
            explanation: "Ship the composer chip",
            plan: [
              { step: "Read the shell", status: "complete" },
              { step: "Move the checklist", status: "in_progress" },
              { step: "Cover with a test", status: "pending" },
            ],
          }),
        },
      },
      {
        type: "tool_result",
        session_id: "s1",
        ts: "2026-01-01T00:00:03Z",
        payload: {
          id: "p1",
          name: "update_plan",
          content: "Ship the composer chip\n\n1. [complete] Read the shell\n2. [in_progress] Move the checklist\n3. [pending] Cover with a test\n",
        },
      },
      {
        type: "plan",
        session_id: "s1",
        ts: "2026-01-01T00:00:03Z",
        payload: {
          name: "update_plan",
          id: "p1",
          text: "Ship the composer chip\n\n1. [complete] Read the shell\n2. [in_progress] Move the checklist\n3. [pending] Cover with a test\n",
        },
      },
      { type: "assistant", session_id: "s1", ts: "2026-01-01T00:00:04Z", payload: { text: "working the list", id: "s1:r2" } },
    ],
  });
  await page.goto("/");
  const chip = page.getByTestId("task-plan");
  await expect(chip).toBeVisible();
  await expect(page.getByRole("button", { name: "Show plan" })).toBeVisible();
  await expect(chip).toContainText("Move the checklist");
  await expect(chip).toContainText("1/3");
  await expect(page.getByTestId("artifact-card")).toHaveCount(0);
  await expect(page.getByTestId("process-summary")).toContainText("Updated the plan");
  await expect(chip.getByText("Cover with a test")).toHaveCount(0);
  await expect(page.getByTestId("conversation-column").getByText("Ship the composer chip")).toHaveCount(0);
  await page.getByRole("button", { name: "Show plan" }).click();
  await expect(chip.getByText("Cover with a test")).toBeVisible();
  await page.getByRole("button", { name: "Hide plan" }).click();
  await expect(page.getByRole("button", { name: "Show plan" })).toBeVisible();
  await expect(chip.getByText("Cover with a test")).toHaveCount(0);
});

test("task plan leaves the composer when the next user turn starts", async ({ page }) => {
  await mockApi(page, "C:/tmp/ws", {
    sessions: [{ id: "s1", title: "Demo thread", workspace: "C:/tmp/ws" }],
    events: [
      { type: "user", session_id: "s1", ts: "2026-01-01T00:00:00Z", payload: { text: "first" } },
      {
        type: "tool_call",
        session_id: "s1",
        ts: "2026-01-01T00:00:01Z",
        payload: {
          id: "p1",
          name: "update_plan",
          arguments: JSON.stringify({ plan: [{ step: "Old work", status: "in_progress" }] }),
        },
      },
      {
        type: "plan",
        session_id: "s1",
        ts: "2026-01-01T00:00:02Z",
        payload: { name: "update_plan", id: "p1", text: "1. [in_progress] Old work\n" },
      },
      { type: "user", session_id: "s1", ts: "2026-01-01T00:00:10Z", payload: { text: "second" } },
    ],
  });
  await page.goto("/");
  await expect(page.getByTestId("task-plan")).toHaveCount(0);
  await expect(page.getByTestId("conversation-column").getByText("second")).toBeVisible();
});

test("appearance palettes keep light off paper yellow by default", async ({ page }) => {
  await mockApi(page, "C:/tmp/ws");
  await page.goto("/");
  await page.getByRole("button", { name: "Settings", exact: true }).click();
  await page.getByRole("button", { name: "Appearance" }).click();
  await expect(page.getByRole("button", { name: /Neutral/ })).toBeVisible();
  await expect(page.getByRole("button", { name: /Paper/ })).toBeVisible();
  await expect(page.getByRole("button", { name: /Mist/ })).toBeVisible();
  await expect(page.getByRole("button", { name: /Ink/ })).toBeVisible();
  await page.getByRole("button", { name: "Light", exact: true }).click();
  await expect(page.locator("html")).toHaveClass(/light/);
  await expect(page.locator("html")).toHaveAttribute("data-palette", "neutral");
  await page.getByRole("button", { name: /Paper/ }).click();
  await expect(page.locator("html")).toHaveAttribute("data-palette", "paper");
  await page.getByRole("button", { name: /Neutral/ }).click();
  await expect(page.locator("html")).toHaveAttribute("data-palette", "neutral");
  await page.getByRole("button", { name: "Dark", exact: true }).click();
  await expect(page.locator("html")).toHaveClass(/dark/);
  await expect(page.locator("html")).toHaveAttribute("data-palette", "ink");
  await expect(page.getByRole("button", { name: /Neutral/ })).toHaveAttribute("aria-pressed", "true");
  await page.getByRole("button", { name: /Slate/ }).click();
  await expect(page.locator("html")).toHaveAttribute("data-palette", "slate");
  await expect(page.locator("html")).toHaveClass(/dark/);
});

test("advanced diagnostics is not the journal", async ({ page }) => {
  await mockApi(page, "C:/tmp/ws");
  await page.goto("/");
  await page.getByRole("button", { name: "Settings", exact: true }).click();
  await page.getByRole("button", { name: "Advanced" }).click();
  await expect(page.getByRole("heading", { name: "Diagnostics" })).toBeVisible();
  await expect(page.getByRole("heading", { name: "Journal" })).toBeVisible();
  await expect(page.getByTestId("diagnostics-tail")).toContainText("boot");
  await expect(page.getByTestId("journal-tail")).toContainText("checkout");
  await expect(page.getByText("Journal tail. The agent cannot edit this file.")).toHaveCount(0);
});

test("companion surface does not render the workstation", async ({ page }) => {
  await mockApi(page, "C:/tmp/ws");
  await page.goto("/?surface=companion");
  await expect(page.getByTestId("companion-stage")).toBeVisible();
  await expect(page.getByTestId("companion-bubble")).toBeAttached();
  await expect(page.locator(".companion-close")).toHaveCount(0);
  await expect(page.getByRole("button", { name: "New chat", exact: true })).toHaveCount(0);
  await expect(page.getByRole("textbox", { name: "Message" })).toHaveCount(0);
  await expect(page.getByRole("button", { name: "Settings", exact: true })).toHaveCount(0);
});

test("companion bubble shows loading while a turn is running", async ({ page }) => {
  await mockApi(page, "C:/tmp/ws", {
    sessions: [{ id: "s1", title: "live", created_at: "2026-01-01T00:00:00Z" }],
    running: true,
  });
  await page.goto("/?surface=companion");
  await expect(page.getByTestId("companion-bubble")).toHaveAttribute("data-loading", "true");
  await expect(page.getByTestId("companion-bubble")).toHaveClass(/is-loading/);
  await expect(page.locator(".companion-dots")).toBeVisible();
});
