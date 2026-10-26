import assert from "node:assert/strict";
import { describe, it } from "node:test";
import { BUILTIN_PALETTES } from "./builtin.ts";
import { contrastOf, hslToHex, veilForReadability } from "./contrast.ts";
import { isUserSkinId } from "./schema.ts";
import { isSafeTokenName, isSafeTokenValue, sanitizeTokenMap } from "./validate.ts";

/**
 * 拥有：内置调色的对比度、令牌注入、用户皮肤 id、壁纸纱和取色结果的约定。
 * 不拥有：上屏和窗口背景。
 */

describe("skin contract", () => {
  /** 六套内置调色都通过对比度。 */
  it("builtin palettes pass contrast", () => {
    for (const [id, tokens] of Object.entries(BUILTIN_PALETTES)) {
      const rep = contrastOf(tokens);
      assert.equal(rep.pass, true, `${id}: ${rep.reason}`);
    }
  });

  /** 拒绝把样式注入写进令牌，表外的名字也不会留下。 */
  it("rejects css injection", () => {
    assert.equal(isSafeTokenValue("color", "red; } body { background: url(https://x)"), false);
    assert.equal(isSafeTokenName("--not-real"), false);
    assert.equal(isSafeTokenName("--background"), true);
    assert.equal(sanitizeTokenMap({ "--background": "#111111", "color": "red" })["--background"], "#111111");
    assert.equal(sanitizeTokenMap({ "--background": "#111111", color: "red" }).color, undefined);
    assert.equal(isSafeTokenValue("color", "var(--x);@import"), false);
  });

  /** 用户皮肤 id 接受普通名字，拒绝内置前缀和路径穿越。 */
  it("user skin ids reject builtins and traversal", () => {
    assert.equal(isUserSkinId("user.abc"), true);
    assert.equal(isUserSkinId("builtin.ink"), false);
    assert.equal(isUserSkinId("../x"), false);
  });

  /** 亮壁纸会加纱，但纱有上限，照片仍然可见。暗壁纸纱更薄。 */
  it("veil raises over a bright wallpaper without banning it", () => {
    const bright = veilForReadability(0.88, 0.012, 0.91, 0.26);
    assert.ok(bright > 0.35, `veil ${bright}`);
    assert.ok(bright <= 0.58, `veil must leave the photo visible: ${bright}`);
    const darkWall = veilForReadability(0.06, 0.012, 0.91, 0.26);
    assert.ok(darkWall < 0.2, `dark wall veil ${darkWall}`);
  });

  /** 从色相取出的十六进制背景和前景能通过对比度。 */
  it("wallpaper extract emits hex tokens that pass contrast", () => {
    const darkBg = hslToHex(32, 10, 11);
    const darkFg = hslToHex(32, 6, 96);
    assert.match(darkBg, /^#[0-9a-f]{6}$/);
    const rep = contrastOf({ "--background": darkBg, "--foreground": darkFg, "--muted": hslToHex(32, 4, 58) });
    assert.equal(rep.pass, true, rep.reason);
  });
});

