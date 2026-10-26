import assert from "node:assert/strict";
import { describe, it } from "node:test";
import { ScrollFollow } from "./scroll-follow.ts";

/**
 * 拥有：跟随意图在离开底部、贴底阈值和原生动画中的约定。
 * 不拥有：真实滚动容器和浏览器的平滑滚动实现。
 */

describe("ScrollFollow", () => {
  /** 读者离开底部后，跟随意图放开。 */
  it("releases follow intent when the reader moves away from the floor", () => {
    const follow = new ScrollFollow(true, 80);
    assert.equal(follow.sample({ top: 0, floor: 400, height: 200 }), false);
    assert.equal(follow.active, false);
  });

  /** 仍在阈值内时保持跟随。 */
  it("keeps follow intent within the threshold", () => {
    const follow = new ScrollFollow(true, 80);
    assert.equal(follow.sample({ top: 330, floor: 400, height: 200 }), true);
    assert.equal(follow.nearBottom({ top: 330, floor: 400, height: 200 }), true);
  });

  /** 原生动画的中途位移仍算跟随，不当作读者自己在滚。 */
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
