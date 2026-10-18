import assert from "node:assert/strict";
import { describe, it } from "node:test";
import { ScrollFollow } from "./scroll-follow.ts";

describe("ScrollFollow", () => {
  it("releases follow intent when the reader moves away from the floor", () => {
    const follow = new ScrollFollow(true, 80);
    assert.equal(follow.sample({ top: 0, floor: 400, height: 200 }), false);
    assert.equal(follow.active, false);
  });

  it("keeps follow intent within the threshold", () => {
    const follow = new ScrollFollow(true, 80);
    assert.equal(follow.sample({ top: 330, floor: 400, height: 200 }), true);
    assert.equal(follow.nearBottom({ top: 330, floor: 400, height: 200 }), true);
  });

  it("does not treat native animation progress as reader movement", () => {
    const follow = new ScrollFollow(true, 80);
    const el = {
      scrollTop: 0,
      scrollTo(opts: { top: number }) {
        this.scrollTop = opts.top;
      },
    } as unknown as HTMLElement;
    follow.toBottom(el, { top: 0, floor: 400, height: 200 }, "smooth");
    assert.equal(follow.animating, true);
    assert.equal(follow.sample({ top: 120, floor: 400, height: 200 }), true);
  });
});
