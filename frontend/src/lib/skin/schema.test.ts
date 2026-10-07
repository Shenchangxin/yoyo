import assert from "node:assert/strict";
import { describe, it } from "node:test";
import { BUILTIN_PALETTES } from "./builtin.ts";
import { contrastOf, hslToHex, veilForReadability } from "./contrast.ts";
import { isUserSkinId } from "./schema.ts";
import { isSafeTokenName, isSafeTokenValue, sanitizeTokenMap } from "./validate.ts";

describe("skin contract", () => {
  it("builtin palettes pass contrast", () => {
    for (const [id, tokens] of Object.entries(BUILTIN_PALETTES)) {
      const rep = contrastOf(tokens);
      assert.equal(rep.pass, true, `${id}: ${rep.reason}`);
    }
  });

  it("rejects css injection", () => {
    assert.equal(isSafeTokenValue("color", "red; } body { background: url(https://x)"), false);
    assert.equal(isSafeTokenName("--not-real"), false);
    assert.equal(isSafeTokenName("--background"), true);
    assert.equal(sanitizeTokenMap({ "--background": "#111111", "color": "red" })["--background"], "#111111");
    assert.equal(sanitizeTokenMap({ "--background": "#111111", color: "red" }).color, undefined);
    assert.equal(isSafeTokenValue("color", "var(--x);@import"), false);
  });

  it("user skin ids reject builtins and traversal", () => {
    assert.equal(isUserSkinId("user.abc"), true);
    assert.equal(isUserSkinId("builtin.ink"), false);
    assert.equal(isUserSkinId("../x"), false);
  });

  it("veil raises over a bright wallpaper without banning it", () => {
    const bright = veilForReadability(0.88, 0.012, 0.91, 0.26);
    assert.ok(bright > 0.35, `veil ${bright}`);
    assert.ok(bright <= 0.58, `veil must leave the photo visible: ${bright}`);
    const darkWall = veilForReadability(0.06, 0.012, 0.91, 0.26);
    assert.ok(darkWall < 0.2, `dark wall veil ${darkWall}`);
  });

  it("wallpaper extract emits hex tokens that pass contrast", () => {
    const darkBg = hslToHex(32, 10, 11);
    const darkFg = hslToHex(32, 6, 96);
    assert.match(darkBg, /^#[0-9a-f]{6}$/);
    const rep = contrastOf({ "--background": darkBg, "--foreground": darkFg, "--muted": hslToHex(32, 4, 58) });
    assert.equal(rep.pass, true, rep.reason);
  });
});

