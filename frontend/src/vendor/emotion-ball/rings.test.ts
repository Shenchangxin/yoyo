import assert from "node:assert/strict";
import { describe, it } from "node:test";
import { readFileSync } from "node:fs";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";
import vm from "node:vm";

const src = readFileSync(join(dirname(fileURLToPath(import.meta.url)), "rings.js"), "utf8");
const ctx = { window: {} as { EB_RINGS?: Rings } };
vm.runInNewContext(src.replace(/\nexport \{\};\s*$/, "\n"), ctx);
const RD = ctx.window.EB_RINGS!;

type Pt = [number, number];
type Rings = {
  HEAD_C: number;
  EXPRESSIONS: [Pt[], Pt[]][];
  MOUTHS: Record<string, Pt[]>;
};

function centroid(ring: Pt[]): { x: number; y: number } {
  let x = 0, y = 0;
  for (const p of ring) { x += p[0]; y += p[1]; }
  return { x: x / ring.length, y: y / ring.length };
}

function bbox(ring: Pt[]): { w: number; h: number } {
  let minX = Infinity, maxX = -Infinity, minY = Infinity, maxY = -Infinity;
  for (const p of ring) {
    if (p[0] < minX) minX = p[0];
    if (p[0] > maxX) maxX = p[0];
    if (p[1] < minY) minY = p[1];
    if (p[1] > maxY) maxY = p[1];
  }
  return { w: maxX - minX, h: maxY - minY };
}

describe("emotion-ball rings", () => {
  it("ships 25 conjugated eye pairs on one horizon", () => {
    assert.equal(RD.EXPRESSIONS.length, 25);
    let sumDy = 0;
    for (let i = 0; i < RD.EXPRESSIONS.length; i++) {
      const [left, right] = RD.EXPRESSIONS[i]!;
      assert.equal(left.length, 48, `ring ${i} left`);
      assert.equal(right.length, 48, `ring ${i} right`);
      const dy = Math.abs(centroid(left).y - centroid(right).y);
      assert.ok(dy < 0.35, `ring ${i} |dY|=${dy.toFixed(3)}`);
      sumDy += dy;
    }
    assert.ok(sumDy / RD.EXPRESSIONS.length < 0.2);
  });

  it("keeps the rest pair as a large horizontal oval", () => {
    const [left] = RD.EXPRESSIONS[0]!;
    const box = bbox(left);
    assert.ok(box.w > 36, `rest width ${box.w.toFixed(1)}`);
    assert.ok(box.h > 22, `rest height ${box.h.toFixed(1)}`);
    assert.ok(box.w / box.h > 1.15, `rest aspect ${(box.w / box.h).toFixed(3)}`);
  });

  it("exposes designed mouth slots in local coordinates", () => {
    const slots = ["smile", "grin", "o", "flat", "frown", "wavy", "pout", "open", "dot"];
    for (const slot of slots) {
      const ring = RD.MOUTHS[slot];
      assert.ok(ring && ring.length >= 16, slot);
      const c = centroid(ring);
      assert.ok(Math.abs(c.x) < 0.6, `${slot} cx`);
      assert.ok(Math.abs(c.y) < 8, `${slot} cy`);
    }
  });
});
