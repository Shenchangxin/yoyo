import assert from "node:assert/strict";
import { describe, it } from "node:test";
import { readFileSync } from "node:fs";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";
import vm from "node:vm";

const src = readFileSync(join(dirname(fileURLToPath(import.meta.url)), "emotions.js"), "utf8");
const ctx = { window: {} as { EMOTION_SEED?: { id: string; group: string; anims?: { type: string }[] }[] } };
vm.runInNewContext(src.replace(/\nexport \{\};\s*$/, "\n"), ctx);
const seed = ctx.window.EMOTION_SEED || [];

describe("emotion catalog", () => {
  it("ships fifty unique faces across the reserved slots", () => {
    assert.equal(seed.length, 50);
    const ids = seed.map((e) => e.id);
    assert.equal(new Set(ids).size, 50);
    for (const id of [
      "00", "07", "08", "09", "10", "21", "22", "29", "30", "41", "42", "49",
    ]) {
      assert.ok(ids.includes(id), id);
    }
  });

  it("only uses engine-known animator types", () => {
    const known = new Set(["sine", "pulse", "jitter", "scan", "glance", "blink", "ease", "nod", "sway", "hold"]);
    for (const emo of seed) {
      for (const anim of emo.anims || []) {
        assert.ok(known.has(anim.type), `${emo.id} ${anim.type}`);
      }
    }
  });
});
