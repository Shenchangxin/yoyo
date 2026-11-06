/**
 * 本文件拥有 videoItemOf 的断言。不拥有事件总线，也不改服务端。
 */
import assert from "node:assert/strict";
import { describe, it } from "node:test";
import { videoItemOf } from "./video-item.ts";

describe("videoItemOf", () => {
  /** 同进程桌面：Wails 信封里是 NoticeOf，种类和载荷在 event 里。 */
  it("reads the nested notice the in-process desktop sends", () => {
    const got = videoItemOf({
      name: "yoyo:item",
      data: {
        seq: 3,
        session_id: "s1",
        type: "system",
        event: { type: "system", source: "video", item_kind: "video_route", payload: { pane: "studio", id: "p1" } },
      },
    });
    assert.equal(got.kind, "video_route");
    assert.equal(got.payload.pane, "studio");
    assert.equal(got.payload.id, "p1");
  });

  /** 隔离进程：item_kind 和 payload 在顶层。 */
  it("reads the flat params the isolated worker sends", () => {
    const got = videoItemOf({ name: "yoyo:item", data: { seq: 4, type: "system", item_kind: "cut_state", payload: { id: "c1" } } });
    assert.equal(got.kind, "cut_state");
    assert.equal(got.payload.id, "c1");
  });

  /** 没有内联事件的通知不算视频条目。 */
  it("returns an empty kind for notices without an event", () => {
    assert.equal(videoItemOf({ name: "yoyo:item", data: { seq: 5, type: "assistant" } }).kind, "");
    assert.equal(videoItemOf(null).kind, "");
  });
});
