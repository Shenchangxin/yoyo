import { expect, test } from "@playwright/test";
import { mockApi } from "./mock";

test("opens the agent without a setup dialog", async ({ page }) => {
  await mockApi(page, "");
  await page.goto("/");
  await expect(page.getByRole("dialog", { name: /Set up this workstation/i })).toHaveCount(0);
  await expect(page.getByRole("button", { name: "New chat" })).toBeVisible();
  await expect(page.getByRole("textbox", { name: "Message" })).toBeEnabled();
});

test("new chat is available on first launch", async ({ page }) => {
  await mockApi(page, "");
  await page.goto("/");
  await expect(page.getByRole("button", { name: "New chat" })).toBeVisible();
});

test("command palette opens", async ({ page }) => {
  await mockApi(page, "C:/tmp/ws");
  await page.goto("/");
  await expect(page.getByRole("button", { name: "New chat" })).toBeVisible();
  await page.keyboard.press("Control+k");
  await expect(page.getByPlaceholder("Jump to a thread or action…")).toBeVisible();
});

test("review sheet appears on a narrow viewport", async ({ page }) => {
  await page.setViewportSize({ width: 900, height: 800 });
  await mockApi(page, "C:/tmp/ws");
  await page.goto("/");
  await expect(page.getByRole("dialog", { name: /Set up this workstation/i })).toHaveCount(0);
  await expect(page.getByRole("button", { name: "New chat" })).toBeVisible();
  await expect(page.getByRole("tablist", { name: "Review" })).toBeVisible();
  await expect(page.getByRole("button", { name: "Toggle review" })).toHaveAttribute("aria-pressed", "true");
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

test("thread context menu can pin a chat", async ({ page }) => {
  await mockApi(page, "C:/tmp/ws", {
    sessions: [{ id: "s1", title: "Demo thread", workspace: "C:/tmp/ws" }],
  });
  await page.goto("/");
  await page.getByRole("button", { name: /Demo thread/ }).hover();
  await page.getByRole("button", { name: "Chat actions" }).click();
  await page.getByRole("menuitem", { name: "Pin" }).click();
  await expect(page.getByRole("button", { name: /Demo thread/ })).toBeVisible();
  await page.getByRole("button", { name: "Chat actions" }).click();
  await expect(page.getByRole("menuitem", { name: "Unpin" })).toBeVisible();
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

test("review pane has diff files and trace", async ({ page }) => {
  await mockApi(page, "C:/tmp/ws");
  await page.goto("/");
  await expect(page.getByRole("tablist", { name: "Review" })).toBeVisible();
  await expect(page.getByRole("tab", { name: /^Diff/ })).toBeVisible();
  await expect(page.getByRole("tab", { name: /^Files/ })).toBeVisible();
  await expect(page.getByRole("tab", { name: /^Trace/ })).toBeVisible();
  await expect(page.getByRole("tab", { name: /Context/ })).toHaveCount(0);
  await expect(page.getByRole("tab", { name: /Ask/ })).toHaveCount(0);
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

test("conversation uses the pane while composer stays a reading column", async ({ page }) => {
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
  await expect(thread).toBeVisible();
  await expect(composer).toBeVisible();
  const a = await thread.boundingBox();
  const b = await composer.boundingBox();
  expect(a && b).toBeTruthy();
  expect(a!.width).toBeGreaterThan(b!.width + 24);
  expect(b!.width).toBeLessThanOrEqual(768);
});

test("insert mention keeps the popup open and pins a file token", async ({ page }) => {
  await mockApi(page, "C:/tmp/ws", {
    sessions: [{ id: "s1", title: "Demo thread", workspace: "C:/tmp/ws" }],
  });
  await page.goto("/");
  const box = page.getByRole("textbox", { name: "Message" });
  await expect(box).toBeEnabled({ timeout: 15_000 });
  await page.getByRole("button", { name: "Insert mention" }).click();
  await expect(page.getByRole("listbox", { name: "Mentions" })).toBeVisible();
  await page.getByRole("option", { name: /@file:/ }).first().click();
  await expect(page.getByRole("option", { name: /@file:src\/main.go/ })).toBeVisible();
  await page.getByRole("option", { name: /@file:src\/main.go/ }).click();
  await expect(page.getByTestId("mention-chip")).toHaveAttribute("aria-label", "@file:src/main.go");
  await expect(page.getByTestId("mention-chip")).toContainText("main.go");
  await expect(page.getByRole("button", { name: "More" })).toHaveCount(0);
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
  await expect(card.getByText("Artifact")).toBeVisible();
  await expect(card.getByText("reports/week.docx")).toBeVisible();
  await expect(card.getByRole("button", { name: "Open in Review" })).toBeVisible();
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
  await expect(chip).toContainText("Move the checklist");
  await expect(chip).toContainText("1/3");
  await expect(page.getByTestId("artifact-card")).toHaveCount(0);
  await expect(page.getByTestId("process-summary")).toContainText("Updated the plan");
  await expect(page.getByTestId("conversation-column").getByText("Ship the composer chip")).toHaveCount(0);
  await page.getByRole("button", { name: "Hide plan" }).click();
  await expect(page.getByRole("button", { name: "Show plan" })).toBeVisible();
  await expect(chip.getByText("Cover with a test")).toHaveCount(0);
  await page.getByRole("button", { name: "Show plan" }).click();
  await expect(chip.getByText("Cover with a test")).toBeVisible();
});
