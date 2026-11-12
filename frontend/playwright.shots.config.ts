/**
 * 拥有：截图脚本的 Playwright 配置。和 e2e 分开：自己的目录、自己的端口，用本机已装的 Edge，不下载浏览器。
 * 不拥有：断言。shots/ 里的脚本只出图，不算测试，CI 不跑它。
 *
 * 用法：npx playwright test -c playwright.shots.config.ts
 * 图写到 SHOTS_OUT 指定的目录，默认 shots/out。不要指到 test-results 里，Playwright 每次开跑都会清空它。
 *
 * 没装 Playwright 自带的 Chromium 时，也可以借它用本机的 Edge 跑 e2e：
 * SHOTS_DIR=./e2e npx playwright test -c playwright.shots.config.ts buddy.spec.ts
 */

import { defineConfig } from "@playwright/test";

const port = Number(process.env.SHOTS_PORT) || 4174;

export default defineConfig({
  testDir: process.env.SHOTS_DIR || "./shots",
  timeout: process.env.SHOTS_DIR ? 30_000 : 90_000,
  fullyParallel: false,
  workers: process.env.SHOTS_DIR ? 4 : 1,
  reporter: "line",
  use: {
    baseURL: `http://127.0.0.1:${port}`,
    channel: process.env.SHOTS_CHANNEL || "msedge",
    viewport: { width: 1440, height: 900 },
    deviceScaleFactor: 1,
  },
  webServer: {
    command: `npx vite --host 127.0.0.1 --port ${port} --strictPort`,
    url: `http://127.0.0.1:${port}`,
    reuseExistingServer: true,
    timeout: 120_000,
    env: { ...process.env, VITE_E2E: "1" },
  },
});
