import assert from "node:assert/strict";
import { describe, it } from "node:test";
import { readFileSync } from "node:fs";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";

const root = dirname(fileURLToPath(import.meta.url));

describe("presence letterhead", () => {
  it("renders a static rest face and never portals the live engine onto a turn", () => {
    const stamp = readFileSync(join(root, "PresenceRuntime.tsx"), "utf8");
    assert.match(stamp, /data-testid="presence-stamp"/);
    assert.match(stamp, /presence-stamp-eye/);
    assert.match(stamp, /rx="2\.15" ry="4\.4"/);
    assert.doesNotMatch(stamp, /presence-stamp-spark/);
    assert.match(stamp, /presence-stamp-body/);
    assert.doesNotMatch(stamp, /stopOpacity/);
    assert.doesNotMatch(stamp, /eyeScale:\s*glyph \? 1\.6/);

    const transcript = readFileSync(join(root, "../Transcript.tsx"), "utf8");
    assert.match(transcript, /<PresenceStamp size=\{22\}/);
    assert.doesNotMatch(transcript, /PresenceAnchor id="avatar"/);

    const process = readFileSync(join(root, "../transcript/ProcessGroup.tsx"), "utf8");
    assert.doesNotMatch(process, /PresenceAnchor id="process"/);
    assert.match(process, /pulse-dot/);
  });
});
