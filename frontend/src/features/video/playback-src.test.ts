import assert from "node:assert/strict";
import { describe, it } from "node:test";
import { playbackSrc } from "./playback-src.ts";

const media = "http://127.0.0.1:43123/media/abc123";

describe("playbackSrc", () => {
  it("keeps the loopback url on an http dev page", () => {
    assert.equal(playbackSrc(media, { protocol: "http:", hostname: "localhost" }), media);
  });

  it("uses the app origin when the webview is https or wails", () => {
    assert.equal(playbackSrc(media, { protocol: "https:", hostname: "wails.localhost" }), "/media/abc123");
    assert.equal(playbackSrc(media, { protocol: "http:", hostname: "wails.localhost" }), "/media/abc123");
  });

  it("leaves other urls alone", () => {
    assert.equal(playbackSrc("", { protocol: "https:", hostname: "wails.localhost" }), "");
    assert.equal(playbackSrc("https://cdn.example/a.mp4", { protocol: "https:", hostname: "wails.localhost" }), "https://cdn.example/a.mp4");
  });
});
