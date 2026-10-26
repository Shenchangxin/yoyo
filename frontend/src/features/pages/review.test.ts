/** 拥有页面审阅文本解析的断言。不拥有卡片。不改服务端。 */
import assert from "node:assert/strict";
import { describe, it } from "node:test";
import { parsePageReviewBlob } from "./review.ts";

describe("parsePageReviewBlob", () => {
  /** 读出等待审阅的载荷，后面多余的 JSON 不管。 */
  it("reads the awaiting payload and ignores trailing JSON", () => {
    const payload = {"review_id":"rev-abc","hash":"deadbeef","title":"Note","kind":"page.save"};
    const blob = `awaiting page review ${JSON.stringify(payload)} {"noise":{"nested":true}}`;
    const got = parsePageReviewBlob(blob);
    assert.equal(got?.id, "rev-abc");
    assert.equal(got?.hash, "deadbeef");
    assert.equal(got?.title, "Note");
  });

  /** 没有审阅编号就空。 */
  it("returns null without a review id", () => {
    assert.equal(parsePageReviewBlob("not a review"), null);
  });
});
