import assert from "node:assert/strict";
import { describe, it } from "node:test";
import type { Thread, ThreadChannel } from "./protocol.ts";
import { channelForSurface, threadChannel } from "./protocol.ts";
import { pickChannelThread, selectSurfaceThread } from "./pick-channel.ts";

/**
 * 拥有：换模块时不能把别的模块的对话带过来。
 * 不拥有：侧栏和发送。
 */

function thread(id: string, channel: ThreadChannel): Thread {
  return { id, channel, title: id, workspace: "", toolRoot: "", createdAt: "" } as Thread;
}

describe("channel identity", () => {
  /** 学习表面用学习频道，视频用视频频道，其余用会话频道。 */
  it("assigns a channel per surface", () => {
    assert.equal(channelForSurface("pages"), "learn");
    assert.equal(channelForSurface("video"), "video");
    assert.equal(channelForSurface("agent"), "agent");
    assert.equal(channelForSurface("skills"), "agent");
    assert.equal(threadChannel({ channel: "learn" }), "learn");
    assert.equal(threadChannel({ channel: "video" }), "video");
    assert.equal(threadChannel({ channel: "agent" }), "agent");
  });
});

describe("pickChannelThread", () => {
  /** 从会话模块进入学习时，要回到上次的学习对话，而不是任意一条，更不能留在会话对话上。 */
  it("restores the remembered learn thread instead of the agent thread", () => {
    const agent = thread("agent-1", "agent");
    const learnA = thread("learn-a", "learn");
    const learnB = thread("learn-b", "learn");
    const picked = pickChannelThread([agent, learnA, learnB], "learn", "learn-b", agent);
    assert.equal(picked?.id, "learn-b");
  });

  /** 已经停在这条通道上时，刷新不能把它换成记住的另一条。 */
  it("keeps the current thread when the channel already matches", () => {
    const learnA = thread("learn-a", "learn");
    const learnB = thread("learn-b", "learn");
    const picked = pickChannelThread([learnA, learnB], "learn", "learn-a", learnB);
    assert.equal(picked?.id, "learn-b");
  });

  /** 回到会话模块时用记住的会话对话。 */
  it("restores the agent thread when leaving learn", () => {
    const agent = thread("agent-1", "agent");
    const other = thread("agent-2", "agent");
    const learn = thread("learn-a", "learn");
    const picked = pickChannelThread([other, agent, learn], "agent", "agent-1", learn);
    assert.equal(picked?.id, "agent-1");
  });
});

describe("selectSurfaceThread", () => {
  /** 子模块已经钉住一条学习对话时，不能被上次随便记下的另一条盖掉。 */
  it("keeps the held learn thread instead of the remembered one", () => {
    const agent = thread("agent-1", "agent");
    const today = thread("learn-today", "learn");
    const path = thread("learn-path", "learn");
    const picked = selectSurfaceThread([agent, today, path], "learn", "learn-today", agent, "learn-path");
    assert.equal(picked?.id, "learn-path");
  });

  /** 钉住的会话还没写进列表时，不能把它丢掉。 */
  it("keeps a held learn thread that is not in the list yet", () => {
    const path = thread("learn-path", "learn");
    const picked = selectSurfaceThread([thread("agent-1", "agent")], "learn", "", null, "learn-path");
    assert.equal(picked, null);
    const kept = selectSurfaceThread([thread("agent-1", "agent")], "learn", "learn-today", path, "learn-path");
    assert.equal(kept?.id, "learn-path");
  });
});
