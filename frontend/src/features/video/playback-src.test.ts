/**
 * 本文件拥有 playbackSrc 的断言。不拥有播放器，也不改服务端。
 */
import assert from "node:assert/strict";
import { describe, it } from "node:test";
import { playbackSrc } from "./playback-src.ts";

const media = "http://127.0.0.1:43123/media/abc123";

describe("playbackSrc", () => {
  /** 普通 http 开发页保留回环地址。 */
  it("keeps the loopback url on an http dev page", () => {
    assert.equal(playbackSrc(media, { protocol: "http:", hostname: "localhost" }), media);
  });

  /** https 或 wails 页面改成站内 /media 路径。 */
  it("uses the app origin when the webview is https or wails", () => {
    assert.equal(playbackSrc(media, { protocol: "https:", hostname: "wails.localhost" }), "/media/abc123");
    assert.equal(playbackSrc(media, { protocol: "http:", hostname: "wails.localhost" }), "/media/abc123");
  });

  /** 空地址和其他来源的地址原样留下。 */
  it("leaves other urls alone", () => {
    assert.equal(playbackSrc("", { protocol: "https:", hostname: "wails.localhost" }), "");
    assert.equal(playbackSrc("https://cdn.example/a.mp4", { protocol: "https:", hostname: "wails.localhost" }), "https://cdn.example/a.mp4");
  });
});
