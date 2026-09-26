import assert from "node:assert/strict";
import { describe, it } from "node:test";
import { presenceLine, presenceOf, wanderEmotion, type PresenceInput } from "./director.ts";
import type { Item } from "../../lib/protocol.ts";

function item(partial: Partial<Item> & Pick<Item, "type">): Item {
  return {
    key: partial.key || "k",
    type: partial.type,
    sessionId: "s1",
    source: partial.source || "runtime",
    ts: partial.ts || new Date().toISOString(),
    text: partial.text || "",
    name: partial.name || "",
    delta: !!partial.delta,
    payload: partial.payload || {},
  };
}

function base(over: Partial<PresenceInput> = {}): PresenceInput {
  return { booted: true, running: false, items: [], approvals: 0, now: 1_000_000, ...over };
}

describe("presenceOf", () => {
  it("boots before the workstation is ready", () => {
    assert.equal(presenceOf(base({ booted: false })).emotionId, "05");
  });

  it("prefers voice listening over tools", () => {
    assert.equal(
      presenceOf(base({
        running: true,
        voicePhase: "listening",
        items: [item({ type: "tool_call", name: "bash", payload: { id: "1" } })],
      })).emotionId,
      "35",
    );
  });

  it("maps thinking, search, shell, and streaming reply", () => {
    assert.equal(presenceOf(base({ running: true })).emotionId, "30");
    assert.equal(
      presenceOf(base({
        running: true,
        items: [item({ type: "tool_call", name: "web_search", payload: { id: "1" } })],
      })).emotionId,
      "40",
    );
    assert.equal(
      presenceOf(base({
        running: true,
        items: [item({ type: "tool_call", name: "bash", payload: { id: "1" } })],
      })).emotionId,
      "32",
    );
    assert.equal(
      presenceOf(base({
        running: true,
        items: [item({ type: "assistant", delta: true, text: "hi" })],
      })).emotionId,
      "39",
    );
    assert.equal(presenceOf(base({ voicePhase: "speaking" })).emotionId, "39");
    assert.equal(presenceOf(base({ voicePhase: "thinking" })).emotionId, "30");
    assert.equal(
      presenceOf(base({
        running: true,
        items: [item({ type: "tool_call", name: "read_file", payload: { id: "1" } })],
      })).emotionId,
      "16",
    );
    assert.equal(
      presenceOf(base({
        running: true,
        items: [item({ type: "tool_call", name: "memory_read", payload: { id: "1" } })],
      })).emotionId,
      "37",
    );
    assert.equal(
      presenceOf(base({
        running: true,
        items: [item({ type: "tool_call", name: "connector_read", payload: { id: "1" } })],
      })).emotionId,
      "36",
    );
    assert.equal(
      presenceOf(base({
        running: true,
        items: [item({ type: "tool_call", name: "office_create", payload: { id: "1" } })],
      })).tips,
      "office_create",
    );
  });

  it("flashes receive then done, sleeps when unfocused", () => {
    const now = 5_000_000;
    assert.equal(
      presenceOf(base({
        running: true,
        now,
        items: [item({ type: "user", ts: new Date(now - 200).toISOString(), source: "ui" })],
      })).emotionId,
      "31",
    );
    assert.equal(
      presenceOf(base({
        now,
        items: [item({ type: "turn_end", ts: new Date(now - 400).toISOString() })],
      })).emotionId,
      "33",
    );
    assert.equal(
      presenceOf(base({
        now,
        items: [item({ type: "turn_end", ts: new Date(now - 2500).toISOString() })],
      })).emotionId,
      "10",
    );
    assert.equal(
      presenceOf(base({ windowFocused: false, idleMs: 200_000 })).emotionId,
      "00",
    );
    assert.equal(presenceOf(base()).emotionId, wanderEmotion(1_000_000, 0));
  });

  it("holds approval, errors, module load, and interrupt", () => {
    assert.equal(presenceOf(base({ approvals: 1 })).emotionId, "35");
    assert.equal(
      presenceOf(base({
        items: [item({ type: "error", text: "provider failed", ts: new Date(999_000).toISOString() })],
        now: 1_000_000,
      })).emotionId,
      "34",
    );
    assert.equal(
      presenceOf(base({
        items: [item({ type: "error", text: "request denied by gate", ts: new Date(999_000).toISOString() })],
        now: 1_000_000,
      })).emotionId,
      "38",
    );
    assert.equal(presenceOf(base({ moduleLoading: true })).emotionId, "36");
    assert.equal(
      presenceOf(base({
        interrupted: true,
        now: 1_000_000,
        items: [item({ type: "turn_end", ts: new Date(999_500).toISOString() })],
      })).emotionId,
      "41",
    );
    assert.equal(presenceOf(base({
        running: false,
        now: 1_000_000,
        items: [item({ type: "eval", ts: new Date(999_000).toISOString() })],
      })).emotionId,
      "36",
    );
    assert.equal(
      presenceOf(base({
        running: false,
        now: 1_000_000,
        items: [item({ type: "tool_call", name: "bash", payload: { id: "1" }, ts: new Date(999_400).toISOString() })],
      })).emotionId,
      "41",
    );
  });
});

describe("wanderEmotion", () => {
  it("cycles ambient faces then drowses into sleep", () => {
    assert.equal(wanderEmotion(0, 0), "02");
    assert.equal(wanderEmotion(6200, 0), "03");
    assert.equal(wanderEmotion(6200, 30_000), "04");
    assert.equal(wanderEmotion(0, 60_000), "15");
    assert.equal(wanderEmotion(0, 140_000), "06");
    assert.equal(wanderEmotion(0, 200_000), "00");
  });
});

describe("presenceLine", () => {
  const labels = { idle: "Around", working: "Working", think: "Thinking", eval: "Running eval" };
  it("uses known tips and prefixes live tools", () => {
    assert.equal(presenceLine({ emotionId: "02", live: false, tips: "idle" }, labels), "Around");
    assert.equal(presenceLine({ emotionId: "30", live: true, tips: "think" }, labels), "Thinking");
    assert.equal(presenceLine({ emotionId: "32", live: true, tips: "bash" }, labels), "Working · bash");
    assert.equal(presenceLine({ emotionId: "36", live: true, tips: "eval" }, labels), "Running eval");
  });
});
