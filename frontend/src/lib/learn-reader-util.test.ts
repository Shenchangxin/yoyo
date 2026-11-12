/** 锁住阅读器的纯函数：引文在换行和空格不同的渲染文字里也找得到，重复的引文取离给出位置最近的一处，页码和进度不出界。 */

import test from "node:test";
import assert from "node:assert/strict";
import { clampLocator, contextAround, fileSize, findQuote, readPercent } from "./learn-reader-util.ts";

test("引文按去掉空白后的文字找，返回渲染文字里的起止位置", () => {
  const page = "导数是瞬时变化率，\n也是 切线的斜率。";
  const hit = findQuote(page, "瞬时变化率，也是切线的斜率");
  assert.ok(hit);
  assert.equal(page.slice(hit.start, hit.end), "瞬时变化率，\n也是 切线的斜率");
  assert.equal(findQuote(page, "没有这句"), null);
  assert.equal(findQuote(page, "   "), null);
});

test("同一句话出现两次时取离给出位置最近的一处", () => {
  const page = "这个结论成立。中间隔着很多别的话，很多很多。这个结论成立。";
  const first = findQuote(page, "这个结论成立", 0);
  const last = findQuote(page, "这个结论成立", 1);
  assert.equal(first?.start, 0);
  assert.equal(last?.start, page.lastIndexOf("这个结论成立"));
  assert.equal(findQuote(page, "这个结论成立")?.start, 0);
});

test("前后文各取一小段，贴着边时不出界", () => {
  const text = "0123456789";
  assert.deepEqual(contextAround(text, 4, 6, 3), { prefix: "123", suffix: "678" });
  assert.deepEqual(contextAround(text, 0, 10, 3), { prefix: "", suffix: "" });
});

test("页码夹进范围，进度在 0 到 1 之间", () => {
  assert.equal(clampLocator(0, 5), 1);
  assert.equal(clampLocator(9, 5), 5);
  assert.equal(clampLocator(Number.NaN, 5), 1);
  assert.equal(clampLocator(3.7, 5), 3);
  assert.equal(readPercent(1, 1), 1);
  assert.equal(readPercent(5, 10), 0.5);
  assert.equal(readPercent(99, 10), 1);
});

test("文件大小写成 KB 或 MB", () => {
  assert.equal(fileSize(0), "");
  assert.equal(fileSize(512), "1 KB");
  assert.equal(fileSize(3 * 1024 * 1024), "3.0 MB");
});
