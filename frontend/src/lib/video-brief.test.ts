/**
 * 拥有：简报卡读得出缩进或紧凑的 JSON、确认结果和缺账号；不是简报的正文返回空。
 * 不拥有：卡片的渲染。
 */
import assert from "node:assert/strict";
import { describe, it } from "node:test";
import { parseVideoBrief } from "./video-brief.ts";

const plan = {
  brief: { intent: "film", format: "ranking", subject: "三个天天用的东西", seconds: 30, aspect: "9:16" },
  needs: ["image", "speech"], have: { image: ["Kling"] }, local: ["speech"], missing: [], spend: true, ready: true,
};

describe("video brief", () => {
  /** 紧凑 JSON 加确认行，读出片型、账号、本机兜底和开拍。 */
  it("reads the plan and the confirm line", () => {
    const got = parseVideoBrief(`${JSON.stringify(plan)}\nconfirmed: go\ngo: go — 开拍\nnote: operator answered`);
    assert.ok(got);
    assert.equal(got.format, "ranking");
    assert.deepEqual(got.have.image, ["Kling"]);
    assert.deepEqual(got.local, ["speech"]);
    assert.equal(got.confirmed, "go");
  });

  /** 缩进 JSON 没有确认行时 confirmed 为空，缺账号照常读出。 */
  it("reads indented JSON and a missing account without a card", () => {
    const missing = { ...plan, have: {}, missing: ["image"], spend: false, ready: false };
    const got = parseVideoBrief(JSON.stringify(missing, null, 2));
    assert.ok(got);
    assert.equal(got.confirmed, "");
    assert.deepEqual(got.missing, ["image"]);
    assert.equal(got.ready, false);
  });

  /** 不是 JSON 或没有 intent 时返回空。 */
  it("ignores bodies that are not a brief", () => {
    assert.equal(parseVideoBrief("video_brief: intent must be one of cut"), null);
    assert.equal(parseVideoBrief(JSON.stringify({ brief: {} })), null);
  });
});