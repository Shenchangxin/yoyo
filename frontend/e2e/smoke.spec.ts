import { expect, test } from "@playwright/test";
import { mockApi } from "./mock";

test("send is disabled without a workspace", async ({ page }) => {
  await mockApi(page, "");
  await page.goto("/");
  await expect(page.getByRole("dialog", { name: /Set up this workstation/i })).toBeVisible();
  await page.getByRole("button", { name: "Skip for now" }).click();
  await expect(page.getByRole("textbox", { name: "Message" })).toBeDisabled();
  await expect(page.getByRole("button", { name: "Send" })).toBeDisabled();
});

test("new chat is available after setup skip", async ({ page }) => {
  await mockApi(page, "");
  await page.goto("/");
  await page.getByRole("button", { name: "Skip for now" }).click();
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
  await expect(page.getByRole("button", { name: "Plan", exact: true })).toHaveAttribute("aria-pressed", "true");
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
  await page.getByRole("button", { name: "Provider", exact: true }).click();
  await page.getByRole("radio", { name: "DeepSeek" }).click();
  await expect(page.getByRole("textbox", { name: "Base URL" })).toHaveValue("https://api.deepseek.com");
  await expect(page.getByRole("textbox", { name: "Model" })).toHaveValue("deepseek-chat");
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
  await expect(page.locator(".caret-blink")).toHaveCount(0);
  await expect(page.getByText(/1 tools/)).toBeVisible();
});

test("streaming caret sits on the open assistant", async ({ page }) => {
  await mockApi(page, "C:/tmp/ws", {
    sessions: [{ id: "s1", title: "Demo thread", workspace: "C:/tmp/ws" }],
    running: true,
    events: [
      { type: "user", session_id: "s1", ts: "2026-01-01T00:00:00Z", payload: { text: "first question" } },
      { type: "assistant", session_id: "s1", ts: "2026-01-01T00:00:01Z", payload: { text: "old answer", id: "s1:r1" } },
      { type: "user", session_id: "s1", ts: "2026-01-01T00:00:02Z", payload: { text: "second question" } },
      { type: "assistant", session_id: "s1", ts: "2026-01-01T00:00:03Z", payload: { text: "live answer", id: "s1:r2" } },
    ],
  });
  await page.goto("/");
  await expect(page.getByText("live answer")).toBeVisible();
  const caret = page.locator(".caret-blink");
  await expect(caret).toHaveCount(1);
  const live = page.getByTestId("agent-turn").filter({ hasText: "live answer" });
  const old = page.getByTestId("agent-turn").filter({ hasText: "old answer" });
  await expect(live.locator(".caret-blink")).toHaveCount(1);
  await expect(old.locator(".caret-blink")).toHaveCount(0);
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

test("review pane has diff and files only", async ({ page }) => {
  await mockApi(page, "C:/tmp/ws");
  await page.goto("/");
  await expect(page.getByRole("tablist", { name: "Review" })).toBeVisible();
  await expect(page.getByRole("tab", { name: /^Diff/ })).toBeVisible();
  await expect(page.getByRole("tab", { name: /^Files/ })).toBeVisible();
  await expect(page.getByRole("tab", { name: /Context/ })).toHaveCount(0);
  await expect(page.getByRole("tab", { name: /Ask/ })).toHaveCount(0);
});

test("composer has no steer control", async ({ page }) => {
  await mockApi(page, "C:/tmp/ws", {
    sessions: [{ id: "s1", title: "Demo thread", workspace: "C:/tmp/ws" }],
    running: true,
    events: [{ type: "user", session_id: "s1", ts: "2026-01-01T00:00:00Z", payload: { text: "go" } }],
  });
  await page.goto("/");
  await expect(page.getByRole("button", { name: "Steer" })).toHaveCount(0);
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
  await expect(page.getByRole("button", { name: "Allow", exact: true })).toBeVisible();
  await expect(page.getByRole("button", { name: "Reject" })).toBeVisible();
  await expect(page.getByRole("tab", { name: /Ask/ })).toHaveCount(0);
});
