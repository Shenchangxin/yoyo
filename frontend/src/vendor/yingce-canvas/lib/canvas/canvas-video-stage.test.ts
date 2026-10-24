import assert from "node:assert/strict";
import test from "node:test";

import { canvasVideoStageStyle } from "./canvas-video-stage.ts";

test("canvas video stage uses the viewport's screen box and the painted corner radius", () => {
    const style = canvasVideoStageStyle(
        { left: 140, top: 80, width: 320, height: 180 },
        { left: 40, top: 20, width: 800, height: 600 },
        "12px",
        640,
    );
    assert.equal(style.left, "100px");
    assert.equal(style.top, "60px");
    assert.equal(style.width, "320px");
    assert.equal(style.height, "180px");
    assert.equal(style.borderRadius, "6px");
    assert.equal(style.visibility, "visible");
});

test("canvas video stage hides a box that has left the viewport", () => {
    const style = canvasVideoStageStyle(
        { left: -400, top: 10, width: 200, height: 100 },
        { left: 0, top: 0, width: 300, height: 200 },
        "8px 4px",
        200,
    );
    assert.equal(style.visibility, "hidden");
    assert.equal(style.borderRadius, "8px 4px");
});
