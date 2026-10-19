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
    assert.match(stamp, /rx="10\.8" ry="22\.4"/);
    assert.match(stamp, /viewBox="-15 -15 259 259"/);
    assert.doesNotMatch(stamp, /presence-stamp-spark/);
    assert.match(stamp, /presence-stamp-body/);
    assert.doesNotMatch(stamp, /stopOpacity/);
    assert.doesNotMatch(stamp, /eyeScale:\s*glyph \? 1\.6/);
    assert.match(stamp, /id="composer"/);
    assert.match(stamp, /data-testid="presence-perch"/);

    const transcript = readFileSync(join(root, "../Transcript.tsx"), "utf8");
    assert.match(transcript, /<PresenceStamp size=\{props\.compact \? 32 : 36\}/);
    assert.doesNotMatch(transcript, /PresenceAnchor id="avatar"/);

    const composer = readFileSync(join(root, "../Composer.tsx"), "utf8");
    assert.match(composer, /PresencePerch/);
    assert.match(composer, /const perch = !!props\.perch/);
    assert.match(composer, /perch \? <PresencePerch/);

    const process = readFileSync(join(root, "../transcript/ProcessGroup.tsx"), "utf8");
    assert.doesNotMatch(process, /PresenceAnchor id="process"/);
    assert.match(process, /pulse-dot/);

    const css = readFileSync(join(root, "../../styles.css"), "utf8");
    assert.match(css, /\.companion-bubble-scroll \{[^}]*overflow-y:\s*auto/);
    assert.match(css, /\.companion-bubble-scroll \{[^}]*max-height:\s*6em/);
    assert.match(css, /\.companion-bubble-copy \{[^}]*white-space:\s*normal/);
    assert.match(css, /\.companion-bubble-copy \{[^}]*overflow-wrap:\s*anywhere/);
    assert.match(css, /\.companion-pet-wrap \{[^}]*border-radius:\s*0/);
    assert.match(css, /\.companion-pet-wrap \{[^}]*overflow:\s*visible/);
    assert.doesNotMatch(css, /\.companion-pet-wrap \.presence-engine \{[^}]*animation:\s*presence-float/);
    assert.match(css, /\.companion-bubble \{[^}]*transition:[^}]*420ms/);
    assert.doesNotMatch(css, /\.companion-bubble::after/);
    assert.match(css, /\.creation-composer-with-perch/);
    assert.match(css, /\.creation-message-stamp/);

    const app = readFileSync(join(root, "../../App.tsx"), "utf8");
    assert.match(app, /perch=\{!home && copilotItems\.length === 0\}/);

    const creation = readFileSync(join(root, "../../vendor/yingce-canvas/pages/create/creation-workspace.tsx"), "utf8");
    assert.match(creation, /PresencePerch/);
    assert.match(creation, /variant === "empty" \? <PresencePerch/);
    assert.match(creation, /<PresenceStamp size=\{36\}/);
    assert.doesNotMatch(creation, /creation-message-mark><Clapperboard/);

    const welcome = readFileSync(join(root, "../../vendor/yingce-canvas/components/canvas/canvas-agent-welcome.tsx"), "utf8");
    assert.match(welcome, /<PresenceStamp size=\{72\} live/);

    const canvasEntry = readFileSync(join(root, "../../vendor/yingce-canvas/components/canvas/canvas-short-drama-entry.tsx"), "utf8");
    assert.match(canvasEntry, /<PresenceStamp size=\{96\} live/);
    assert.doesNotMatch(canvasEntry, /CanvasEntryFilmMark/);
  });
});
