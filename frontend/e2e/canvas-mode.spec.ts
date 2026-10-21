import { expect, test, type Page } from "@playwright/test";
import { mockApi } from "./mock";

function island(page: Page) {
  return page.getByTestId("canvas-studio-shell");
}

async function openVideo(page: Page) {
  await page.getByRole("button", { name: "Video" }).click();
}

async function openVideoChat(page: Page) {
  await openVideo(page);
  await expect(page.getByTestId("create-desk")).toBeVisible();
}

test("infinite canvas opens from the video rail", async ({ page }) => {
  await mockApi(page, "C:/tmp/ws");
  await page.goto("/");
  await expect(page.getByRole("button", { name: "Video" })).toBeVisible();
  await openVideo(page);
  await page.getByTestId("video-shell-canvas").click();
  await expect(island(page)).toBeVisible();
  await expect(page.getByText("创作工作台")).toHaveCount(0);
});

test("opening the canvas board mounts the canvas studio", async ({ page }) => {
  await mockApi(page, "C:/tmp/ws");
  await page.goto("/");
  await openVideo(page);
  await page.getByTestId("video-shell-canvas").click();
  await expect(island(page)).toBeVisible();
  await expect(page.getByText("创作工作台")).toHaveCount(0);
});

test("drama workshop still opens from the same switch", async ({ page }) => {
  await mockApi(page, "C:/tmp/ws");
  await page.goto("/");
  await page.getByRole("button", { name: "Video" }).click();
  await page.getByTestId("video-shell-drama").click();
  await expect(island(page)).toBeVisible();
  await expect(page.getByTestId("drama-agent-page")).toBeVisible({ timeout: 30_000 });
  await expect(page.getByText("当前页面没有正常加载")).toHaveCount(0);
  await expect(page.getByTestId("drama-agent-heading")).toBeInViewport();
  await expect(page.getByTestId("drama-agent-hint")).toBeInViewport();
  await expect(page.getByTestId("drama-path")).toBeInViewport();
  await expect(page.getByTestId("drama-agent-input")).toBeInViewport();
  await expect(page.getByTestId("drama-style-picker")).toBeVisible();
  await expect(page.getByTestId("drama-model-picker")).toBeVisible();
  await expect(page.getByTestId("presence-perch")).toBeVisible();
  await page.getByTestId("drama-style-picker").click();
  await expect(page.getByTestId("drama-style-grid")).toBeVisible();
  await page.getByRole("option", { name: "Anime" }).click();
  await expect(page.getByTestId("drama-style-picker")).toContainText("Anime");
  await expect(page.getByText("Paste a chapter or a whole novel…")).toBeVisible();
  await expect(page.getByTestId("drama-open-board")).toBeInViewport();
  await page.getByTestId("drama-open-board").first().click();
  await expect(page.getByTestId("drama-desk")).toBeVisible();
  await expect(page.getByTestId("drama-studio")).toBeVisible();
  await expect(page.getByTestId("drama-agent-page")).toBeVisible();
  await expect(page.getByTestId("drama-phase-script")).toBeVisible();
  await expect(page.getByTestId("drama-phase-cast")).toBeVisible();
  await expect(page.getByTestId("drama-phase-board")).toBeVisible();
  await expect(page.getByTestId("drama-phase-cut")).toBeVisible();
  await page.getByTestId("drama-close-board").click();
  await expect(page.getByTestId("drama-studio")).toHaveCount(0);
  await expect(page.getByTestId("drama-agent-page")).toBeVisible();
});

test("video rail hosts the canvas workspace shell instead of the inner create workbench", async ({ page }) => {
  test.setTimeout(60_000);
  await mockApi(page, "C:/tmp/ws");
  await page.goto("/");
  await page.getByRole("button", { name: "Video" }).click();
  await expect(page.getByTestId("video-shell-nav")).toBeVisible();
  await expect(page.getByTestId("video-shell-create")).toBeVisible();
  await expect(page.getByTestId("video-shell-studio")).toBeVisible();
  await expect(page.getByTestId("create-desk")).toBeVisible({ timeout: 15_000 });
  await expect(page.getByTestId("empty-turn")).toBeVisible();
  await expect(page.getByRole("heading", { name: "Start with a clip." })).toBeVisible();
  await expect(page.getByTestId("creation-handoff")).toBeVisible();
  await expect(page.getByTestId("create-import")).toBeVisible();
  await expect(page.getByRole("button", { name: "Make a short" })).toBeVisible();
  await expect(page.getByTestId("creation-cases")).toBeVisible();
  await expect(page.getByTestId("creation-cases-source")).toContainText("yoyo.creation-cases");
  await expect(page.getByTestId("creation-cases-source")).toContainText("22 briefs");
  await expect(page.getByTestId("creation-cases-source")).toContainText("/short-drama-styles");
  const rain = page.getByTestId("creation-case-rain-clinic");
  await expect(rain).toBeVisible();
  await expect(rain).toContainText("cyberpunk-neon.jpg");
  await expect(rain).toContainText("Opens in Shorts");
  await rain.click();
  await expect(page.getByTestId("composer-input")).toContainText("Open Shorts for this brief");
  await expect(page.getByTestId("composer-input")).toContainText("雨港记忆诊所");
  await expect(page.getByTestId("profile-chip")).toHaveCount(0);
  await expect(page.getByLabel("Where it runs")).toHaveCount(0);
  await expect(page.getByTestId("create-open-footage")).toHaveCount(0);
  await expect(page.getByTestId("footage-desk")).toHaveCount(0);
  await expect(island(page)).toHaveCount(0);
  await expect(page.getByText("创作工作台")).toHaveCount(0);
  await expect(page.getByRole("button", { name: "New chat" })).toHaveCount(0);

  await page.getByTestId("video-shell-drama").click();
  await expect(island(page)).toBeVisible();
  await expect(island(page)).toHaveAttribute("data-pane", "drama");
  await expect(page.getByTestId("drama-agent-page")).toBeVisible({ timeout: 30_000 });
  await expect(page.getByTestId("drama-agent-heading")).toBeVisible();
  await expect(page.getByTestId("drama-agent-hint")).toBeVisible();
  await expect(page.getByTestId("drama-path")).toBeVisible();
  await expect(page.getByTestId("drama-agent-input")).toBeVisible();
  await expect(page.getByTestId("video-mode-switch")).toHaveCount(0);
  await expect(page.locator(".creation-mode-tabs")).toHaveCount(0);
  await expect(page.getByTestId("drama-studio")).toHaveCount(0);
  await expect(page.getByText("创作工作台")).toHaveCount(0);

  await page.getByTestId("video-shell-canvas").click();
  await expect(island(page)).toBeVisible();
  await expect(page.getByText("创作工作台")).toHaveCount(0);
});

test("assets skills plugins and history open from the video rail", async ({ page }) => {
  await mockApi(page, "C:/tmp/ws");
  await page.goto("/");
  await page.getByRole("button", { name: "Video" }).click();
  await expect(page.getByTestId("video-shell-nav")).toBeVisible();
  await expect(page.getByTestId("video-shell-create")).toBeVisible();
  await expect(page.getByTestId("video-shell-assets")).toBeVisible();
  await expect(page.getByTestId("video-shell-skills")).toBeVisible();
  await expect(page.getByTestId("video-shell-plugins")).toBeVisible();
  await expect(page.getByTestId("video-shell-tasks")).toBeVisible();
  await page.getByTestId("video-shell-assets").click();
  await expect(island(page)).toBeVisible();
  const assetsCover = page.locator(".assets-empty-banner-frame img").first();
  await expect(assetsCover).toHaveAttribute("src", /\/short-drama-styles\//);
  await expect.poll(async () => assetsCover.evaluate((el) => (el as HTMLImageElement).naturalWidth)).toBeGreaterThan(40);

  for (const [testid, pane] of [
    ["video-shell-assets", "assets"],
    ["video-shell-skills", "skills"],
    ["video-shell-plugins", "plugins"],
    ["video-shell-tasks", "tasks"],
  ] as const) {
    await page.getByTestId(testid).click();
    await expect(island(page)).toBeVisible();
    await expect(island(page)).toHaveAttribute("data-pane", pane);
    await expect(page.getByText("创作工作台")).toHaveCount(0);
  }

  await page.keyboard.press("ControlOrMeta+k");
  await page.getByTestId("palette-video-assets").click();
  await expect(island(page)).toHaveAttribute("data-pane", "assets");

  await page.keyboard.press("ControlOrMeta+k");
  await page.getByRole("option", { name: "Open Skills" }).click();
  await expect(page.getByTestId("video-shell-nav")).toHaveCount(0);
});

test("studio workbench opens from the video rail", async ({ page }) => {
  await mockApi(page, "C:/tmp/ws");
  await page.goto("/");
  await page.getByRole("button", { name: "Video" }).click();
  await expect(page.getByTestId("video-shell-studio")).toBeVisible();
  await expect(page.getByTestId("video-shell-studio")).toContainText(/Shorts|短片/);
  await page.getByTestId("video-shell-studio").click();
  await expect(page.getByTestId("studio-desk")).toBeVisible();
  await expect(page.getByTestId("studio-workbench")).toBeVisible();
  await expect(page.getByTestId("studio-home")).toBeVisible();
  await expect(page.getByTestId("studio-pane-radar")).toBeVisible();
  await expect(page.getByTestId("studio-pane-topics")).toBeVisible();
  await expect(page.getByTestId("studio-pane-assemble")).toHaveCount(0);
  await expect(page.getByTestId("studio-pane-publish")).toHaveCount(0);
  await expect(page.getByTestId("studio-home-search")).toBeVisible();
  await expect(page.getByTestId("studio-open-settings")).toHaveCount(0);
  await expect(page.getByTestId("studio-new-short")).toBeVisible();
  await expect(page.getByTestId("studio-new-short")).toContainText(/New short|新建短片/);
  await expect(page.getByTestId("studio-delete-current")).toHaveCount(0);
  await expect(page.getByTestId("studio-pipeline")).toBeVisible();
  await expect(page.getByTestId("studio-next")).toBeVisible();
  await expect(page.getByTestId("studio-copilot")).toHaveCount(0);
  await expect(page.getByTestId("video-mode-switch")).toHaveCount(0);
  await expect(page.getByTestId("profile-chip")).toHaveCount(0);
  await expect(page.getByTestId("studio-start")).toHaveCount(0);
  await expect(page.getByTestId("studio-lane-make")).toHaveCount(0);
});

test("studio workbench opens from the composer mode switch", async ({ page }) => {
  await mockApi(page, "C:/tmp/ws");
  await page.goto("/");
  await openVideo(page);
  await page.getByTestId("video-shell-studio").click();
  await expect(page.getByTestId("studio-desk")).toBeVisible();
  await expect(page.getByTestId("studio-workbench")).toBeVisible();
  await expect(page.getByTestId("studio-home")).toBeVisible();
  await expect(page.getByTestId("studio-pane-radar")).toBeVisible();
  await expect(page.getByTestId("studio-pane-topics")).toBeVisible();
  await expect(page.getByTestId("studio-pane-assemble")).toHaveCount(0);
  await expect(page.getByTestId("studio-copilot")).toHaveCount(0);
});

test("studio workbench opens from creation history", async ({ page }) => {
  await mockApi(page, "C:/tmp/ws");
  await page.goto("/");
  await page.getByRole("button", { name: "Video" }).click();
  await page.getByTestId("video-shell-tasks").click();
  await page.getByTestId("video-history-studio-title").fill("春日通勤");
  await page.getByTestId("video-history-new-studio").click();
  await expect(page.getByTestId("studio-desk")).toBeVisible();
  await expect(page.getByTestId("studio-workbench")).toBeVisible();
  await expect(page.getByTestId("studio-workbench-bound")).toHaveText(/春日通勤/);
  await expect(page.getByTestId("studio-id-chip")).toBeVisible();
  await expect(page.getByTestId("studio-copy-id")).toContainText(/studio-1/);
  await expect(page.getByTestId("studio-copilot-bound")).toHaveText(/春日通勤/);
  await expect(page.getByTestId("studio-copilot-id")).toHaveText(/studio-1/);
  await expect(page.getByTestId("studio-copilot-header")).toContainText(/和左侧工台是同一条|Same short as the desk/);
  await expect(page.getByTestId("studio-copilot-next")).toContainText(/下一步|Next/);
  await expect(page.getByTestId("studio-copilot-auth")).toContainText(/完全访问|Full access/);
  await page.getByTestId("studio-pane-assemble").click();
  await expect(page.getByTestId("studio-step-nav")).toBeVisible();
  await expect(page.getByTestId("studio-step-setup")).toBeVisible();
  await expect(page.getByTestId("studio-step-script")).toBeVisible();
  await expect(page.getByTestId("studio-step-audio")).toBeVisible();
  await expect(page.getByTestId("studio-step-visuals")).toBeVisible();
  await expect(page.getByTestId("studio-step-compose")).toBeVisible();
  await expect(page.getByTestId("studio-step-delivery")).toBeVisible();
  await expect(page.getByTestId("studio-ops-tools")).toHaveCount(0);
  await expect(page.getByTestId("studio-collab")).toHaveCount(0);
  await expect(page.getByTestId("studio-cut-ops")).toHaveCount(0);
  await expect(page.getByTestId("studio-new-short")).toHaveCount(0);
  await expect(page.getByTestId("studio-delete-current")).toHaveCount(0);
  await expect(page.getByTestId("studio-brief")).toBeVisible();
  await expect(page.getByTestId("studio-brief")).toContainText(/Who it's for|给谁看/);
  await expect(page.getByTestId("studio-brief")).toContainText(/Length|时长/);
  await expect(page.getByTestId("studio-brief-audience")).toBeVisible();
  await expect(page.getByText(/Audience and length|观众和时长/)).toHaveCount(0);
  await page.getByTestId("studio-step-script").click();
  await expect(page.getByTestId("studio-lines")).toBeVisible();
  await expect(page.getByTestId("studio-lines")).toContainText(/Hook|钩子/);
  await expect(page.getByTestId("studio-lines")).toContainText(/Close|收束/);
  await expect(page.getByTestId("studio-lines")).toContainText(/Tone|语气/);
  await expect(page.getByTestId("studio-lines-hook")).toBeVisible();
  await expect(page.getByTestId("studio-lines-cta")).toBeVisible();
  await expect(page.getByText(/Hook, close, tone|钩子、收束、语气/)).toHaveCount(0);
});

test("assembled short sends to create cut", async ({ page }) => {
  await mockApi(page, "C:/tmp/ws", {
    studios: [{
      id: "studio-cut",
      title: "春日成片",
      status: "assembled",
      gate: "compose",
      furthest_gate: "compose",
      script: "口播稿",
      script_status: "approved",
      result_hash: "cas-assembled",
      storyboard: { shots: [{ id: "s1", image_hash: "img-1" }] },
      delivery: { status: "draft" },
    }],
  });
  await page.goto("/");
  await page.getByRole("button", { name: "Video" }).click();
  await page.getByTestId("video-shell-tasks").click();
  await page.getByTestId("video-project-studio-studio-cut").click();
  await expect(page.getByTestId("studio-desk")).toBeVisible();
  await page.getByTestId("studio-step-compose").click();
  await expect(page.getByTestId("studio-send-to-cut")).toBeVisible();
  await page.getByTestId("studio-send-to-cut").click();
  await expect(page.getByTestId("create-desk")).toBeVisible();
  await expect(page.getByLabel("Message")).toHaveValue(/cut_id=cut-1/);
  await expect(page.getByLabel("Message")).toHaveValue(/cas=cas-assembled/);
});

test("each short keeps its own copilot chat", async ({ page }) => {
  await mockApi(page, "C:/tmp/ws");
  await page.goto("/");
  await page.getByRole("button", { name: "Video" }).click();
  await page.getByTestId("video-shell-studio").click();
  await page.getByTestId("studio-new-short").click();
  await page.getByTestId("studio-create-title").fill("春日通勤");
  await page.getByTestId("studio-create").click();
  await expect(page.getByTestId("studio-workbench-bound")).toHaveText(/春日通勤/);
  await expect(page.getByTestId("studio-copy-session")).toContainText(/s1/);
  await expect(page.getByTestId("studio-new-short")).toHaveCount(0);
  await page.getByTestId("studio-switch-short").click();
  await page.getByTestId("studio-switch-new").click();
  await page.getByTestId("studio-create-title").fill("夜路回访");
  await page.getByTestId("studio-create").click();
  await expect(page.getByTestId("studio-workbench-bound")).toHaveText(/夜路回访/);
  await expect(page.getByTestId("studio-copilot-bound")).toHaveText(/夜路回访/);
  await expect(page.getByTestId("studio-copy-session")).toContainText(/s2/);
  await expect(page.getByTestId("studio-copy-session")).not.toContainText(/s1/);
  await page.getByTestId("studio-switch-short").click();
  await expect(page.getByTestId("studio-switch-studio-1")).toBeVisible();
  await expect(page.getByTestId("studio-switch-studio-2")).toBeVisible();
  await page.getByTestId("studio-switch-studio-1").click();
  await expect(page.getByTestId("studio-workbench-bound")).toHaveText(/春日通勤/);
  await expect(page.getByTestId("studio-copilot-bound")).toHaveText(/春日通勤/);
  await expect(page.getByTestId("studio-copy-session")).toContainText(/s1/);
  await expect(page.getByTestId("studio-copilot-auth")).toContainText(/完全访问|Full access/);
});

test("a topic starts a short project with its own copilot", async ({ page }) => {
  await mockApi(page, "C:/tmp/ws");
  await page.goto("/");
  await page.getByRole("button", { name: "Video" }).click();
  await page.getByTestId("video-shell-studio").click();
  await expect(page.getByTestId("studio-home")).toBeVisible();
  await expect(page.getByTestId("studio-copilot")).toHaveCount(0);
  await page.getByTestId("studio-pane-topics").click();
  await page.getByTestId("studio-topic-topic-1").getByTestId("studio-topic-produce").click();
  await expect(page.getByTestId("studio-workbench-bound")).toHaveText(/春日穿搭/);
  await expect(page.getByTestId("studio-project")).toBeVisible();
  await expect(page.getByTestId("studio-pane-assemble")).toBeVisible();
  await expect(page.getByTestId("studio-pane-radar")).toHaveCount(0);
  await expect(page.getByTestId("studio-copilot")).toBeVisible();
  await expect(page.getByTestId("studio-copilot-bound")).toHaveText(/春日穿搭/);
  await expect(page.getByTestId("studio-copilot-auth")).toContainText(/完全访问|Full access/);
  await page.getByTestId("studio-back-home").click();
  await expect(page.getByTestId("studio-home")).toBeVisible();
  await expect(page.getByTestId("studio-copilot")).toHaveCount(0);
  await expect(page.getByTestId("studio-pane-topics")).toBeVisible();
  await page.getByTestId("studio-pane-topics").click();
  await page.getByTestId("studio-topic-topic-1").getByTestId("studio-topic-open").click();
  await expect(page.getByTestId("studio-project")).toBeVisible();
  await expect(page.getByTestId("studio-home")).toHaveCount(0);
  await expect(page.getByTestId("studio-workbench-bound")).toHaveText(/春日穿搭/);
  await expect(page.getByTestId("studio-pane-assemble")).toBeVisible();
  await expect(page.getByTestId("studio-copilot")).toBeVisible();
});

test("several topics can start shorts together", async ({ page }) => {
  await mockApi(page, "C:/tmp/ws");
  await page.goto("/");
  await page.getByRole("button", { name: "Video" }).click();
  await page.getByTestId("video-shell-studio").click();
  await page.getByTestId("studio-pane-topics").click();
  await page.getByTestId("studio-topic-pick-topic-1").click();
  await page.getByTestId("studio-topic-pick-topic-2").click();
  await expect(page.getByTestId("studio-topic-picked")).toBeVisible();
  await page.getByTestId("studio-new-from-topics").click();
  await expect(page.getByTestId("studio-copilot")).toBeVisible();
  await expect(page.getByTestId("studio-workbench-bound")).toHaveText(/还没想好|Short/);
  await page.getByTestId("studio-back-home").click();
  await page.getByTestId("studio-pane-topics").click();
  await expect(page.getByTestId("studio-topic-topic-1").getByTestId("studio-topic-open")).toBeVisible();
  await expect(page.getByTestId("studio-topic-topic-2").getByTestId("studio-topic-open")).toBeVisible();
});

test("topic score stays on the topic card", async ({ page }) => {
  await mockApi(page, "C:/tmp/ws");
  await page.goto("/");
  await page.getByRole("button", { name: "Video" }).click();
  await page.getByTestId("video-shell-studio").click();
  await page.getByTestId("studio-pane-topics").click();
  const card = page.getByTestId("studio-topic-topic-1");
  await expect(card.getByTestId("studio-topic-scores")).toContainText(/Worth filming|值不值得拍/);
  await card.getByTestId("studio-topic-scores").click();
  await expect(card.getByTestId("studio-topic-score-heat")).toBeVisible();
  await card.getByTestId("studio-topic-score-heat-3").click();
  await expect(page.getByTestId("studio-workbench")).toHaveAttribute("data-pane", "topics");
  await expect(card.getByTestId("studio-topic-more")).toBeVisible();
  await expect(card.getByText(/Scores and persona|打分与人设/)).toHaveCount(0);
});

test("topic schedule stays on topics and does not steal the desk", async ({ page }) => {
  await mockApi(page, "C:/tmp/ws");
  await page.goto("/");
  await page.getByRole("button", { name: "Video" }).click();
  await page.getByTestId("video-shell-studio").click();
  await page.getByTestId("studio-pane-topics").click();
  await expect(page.getByTestId("studio-topic-filters")).toBeVisible();
  await expect(page.getByTestId("studio-topic-topic-1")).toBeVisible();
  const card = page.getByTestId("studio-topic-topic-1");
  await card.getByTestId("studio-topic-more").click();
  await page.getByTestId("studio-topic-schedule").click();
  await expect(page.getByTestId("studio-workbench")).toHaveAttribute("data-pane", "topics");
  await expect(page.getByTestId("studio-id-chip")).toHaveCount(0);
  await page.getByTestId("studio-topic-filter-scheduled").click();
  await expect(page.getByTestId("studio-topic-topic-1")).toHaveAttribute("data-status", "scheduled");
  await expect(page.getByTestId("studio-topic-unschedule")).toBeVisible();
});

test("shorts can be created and a history item can be deleted", async ({ page }) => {
  await mockApi(page, "C:/tmp/ws");
  await page.goto("/");
  await page.getByRole("button", { name: "Video" }).click();
  await page.getByTestId("video-shell-studio").click();
  await expect(page.getByTestId("studio-new-short")).toBeVisible();
  await page.getByTestId("studio-new-short").click();
  await page.getByTestId("studio-create-title").fill("春日通勤");
  await page.getByTestId("studio-create").click();
  await expect(page.getByTestId("studio-workbench-bound")).toHaveText(/春日通勤/);
  await expect(page.getByTestId("studio-new-short")).toHaveCount(0);
  await expect(page.getByTestId("studio-delete-current")).toHaveCount(0);
  await page.getByTestId("studio-switch-short").click();
  await page.getByTestId("studio-switch-new").click();
  await page.getByTestId("studio-create-title").fill("夜路回访");
  await page.getByTestId("studio-create").click();
  await expect(page.getByTestId("studio-workbench-bound")).toHaveText(/夜路回访/);
  await page.getByTestId("studio-switch-short").click();
  await expect(page.getByTestId("studio-switch-new")).toBeVisible();
  await expect(page.getByTestId("studio-delete-studio-1")).toBeVisible();
  await page.getByTestId("studio-delete-studio-1").click();
  await expect(page.getByRole("dialog")).toContainText(/春日通勤/);
  await page.getByRole("dialog").getByRole("button", { name: /Delete short|删除短片/ }).click();
  await expect(page.getByRole("dialog")).toHaveCount(0);
  await expect(page.getByTestId("studio-workbench-bound")).toHaveText(/夜路回访/);
  await page.getByTestId("studio-switch-short").click();
  await expect(page.getByTestId("studio-switch-studio-1")).toHaveCount(0);
  await expect(page.getByTestId("studio-switch-studio-2")).toBeVisible();
  await page.getByTestId("studio-switch-studio-2").click();
  await page.getByTestId("studio-more").click();
  await page.getByTestId("studio-delete-menu").click();
  await expect(page.getByRole("dialog")).toContainText(/夜路回访/);
  await page.getByRole("dialog").getByRole("button", { name: /Delete short|删除短片/ }).click();
  await expect(page.getByTestId("studio-home")).toBeVisible();
  await expect(page.getByTestId("studio-copilot")).toHaveCount(0);
  await expect(page.getByTestId("studio-delete-current")).toHaveCount(0);
  await expect(page.getByTestId("studio-workbench-bound")).toHaveText(/Shorts|短片/);
});

test("creation history can delete a short", async ({ page }) => {
  await mockApi(page, "C:/tmp/ws");
  await page.goto("/");
  await page.getByRole("button", { name: "Video" }).click();
  await page.getByTestId("video-shell-tasks").click();
  await page.getByTestId("video-history-studio-title").fill("春日通勤");
  await page.getByTestId("video-history-new-studio").click();
  await expect(page.getByTestId("studio-desk")).toBeVisible();
  await page.getByTestId("video-shell-tasks").click();
  await expect(page.getByTestId("video-history")).toBeVisible();
  await expect(page.getByTestId("video-project-studio-studio-1")).toBeVisible();
  await page.getByTestId("video-history-delete-studio-1").click();
  await expect(page.getByRole("dialog")).toContainText(/春日通勤/);
  await page.getByRole("dialog").getByRole("button", { name: /Delete short|删除短片/ }).click();
  await expect(page.getByTestId("video-project-studio-studio-1")).toHaveCount(0);
});

test("new canvas appears in creation history and can be opened", async ({ page }) => {
  test.setTimeout(60_000);
  await mockApi(page, "C:/tmp/ws");
  await page.goto("/");
  await page.getByRole("button", { name: "Video" }).click();
  await page.getByTestId("video-shell-tasks").click();
  await expect(page.getByTestId("video-history")).toBeVisible();
  await page.getByTestId("video-history-new-canvas").click();
  await expect(page.getByTestId("video-project-canvas-c1")).toBeVisible();
  await page.getByTestId("video-project-canvas-c1").click();
  await expect(island(page)).toHaveAttribute("data-pane", "canvas");
});
