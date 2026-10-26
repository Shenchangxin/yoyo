/**
 * 拥有：从 harness 载荷里解析指针、评测、谱系和目录，以及下一步该去哪个页签。
 * 不拥有：页签文案，以及评测的执行。
 */

import { asArray, asBool, num, pick, str } from "./normalize";
import type { HarnessTab } from "./protocol";

/** HarnessRefs active、staging、canary、head，以及这四个以外的其余指针。 */
export type HarnessRefs = {
  active: string;
  staging: string;
  canary: string;
  head: string;
  extra: { name: string; hash: string }[];
};

/** HarborMetrics 留入集与留出集的通过数和总数，以及安全失败数。 */
export type HarborMetrics = {
  heldInPass: number;
  heldInTotal: number;
  heldOutPass: number;
  heldOutTotal: number;
  safetyFail: number;
};

/** HarnessNext 下一步页签和动作：演化、评测、检出、阻断或空闲。 */
export type HarnessNext = {
  tab: HarnessTab;
  action: "evolve" | "eval" | "checkout" | "blocked" | "idle";
};

const KNOWN = new Set(["active", "staging", "canary", "head", "HEAD"]);

/** shortHash 去掉 blake3 或 sha256 前缀。长于 n 则截到 n。空哈希返回空串。 */
export function shortHash(hash: string, n = 7): string {
  const s = str(hash).replace(/^(blake3:|sha256:)/i, "");
  if (!s) return "";
  return s.length > n ? s.slice(0, n) : s;
}

/** parseHarnessRefs 读出四个已知指针。active 没有时用 fallbackActive。其余非空指针放进 extra。 */
export function parseHarnessRefs(harness: unknown, fallbackActive = ""): HarnessRefs {
  const raw = (pick(harness, "refs", "Refs") || {}) as Record<string, unknown>;
  const active = firstHash(raw, ["active", "Active"]) || str(pick(harness, "active", "Active"), fallbackActive);
  const staging = firstHash(raw, ["staging", "Staging"]);
  const canary = firstHash(raw, ["canary", "Canary"]);
  const head = firstHash(raw, ["HEAD", "head", "Head"]);
  const extra: { name: string; hash: string }[] = [];
  for (const [k, v] of Object.entries(raw)) {
    if (KNOWN.has(k) || KNOWN.has(k.toLowerCase())) continue;
    const hash = str(v);
    if (hash) extra.push({ name: k, hash });
  }
  return { active, staging, canary, head, extra };
}

/** stagingDirty staging 非空且不等于 active。 */
export function stagingDirty(refs: HarnessRefs): boolean {
  return !!refs.staging && refs.staging !== refs.active;
}

/** canaryDirty canary 非空且不等于 active。 */
export function canaryDirty(refs: HarnessRefs): boolean {
  return !!refs.canary && refs.canary !== refs.active;
}

/** readHarborMetrics 从报告的 metrics 里读出五项计数。报告不是对象时返回空。 */
export function readHarborMetrics(report: unknown): HarborMetrics | null {
  if (report == null || typeof report !== "object") return null;
  const metrics = pick(report, "metrics", "Metrics") || {};
  return {
    heldInPass: num(pick(metrics, "held_in_pass", "HeldInPass")),
    heldInTotal: num(pick(metrics, "held_in_total", "HeldInTotal")),
    heldOutPass: num(pick(metrics, "held_out_pass", "HeldOutPass")),
    heldOutTotal: num(pick(metrics, "held_out_total", "HeldOutTotal")),
    safetyFail: num(pick(metrics, "safety_fail", "SafetyFail")),
  };
}

/** harnessNext 安全失败则停在证明页并阻断。staging 或 canary 与 active 不同且闸门干净，则去晋升并检出。指针脏但闸门不干净则去证明并评测。已有指标则提议页空闲。否则提议页演化。 */
export function harnessNext(refs: HarnessRefs, report: unknown): HarnessNext {
  const metrics = readHarborMetrics(report);
  if (metrics && metrics.safetyFail > 0) return { tab: "prove", action: "blocked" };
  const staging = stagingDirty(refs);
  const canary = canaryDirty(refs);
  if (staging && metrics && gateClean(metrics)) return { tab: "promote", action: "checkout" };
  if (canary && metrics && gateClean(metrics)) return { tab: "promote", action: "checkout" };
  if (staging || canary) return { tab: "prove", action: "eval" };
  if (metrics) return { tab: "propose", action: "idle" };
  return { tab: "propose", action: "evolve" };
}

/** 留入集有题且全过，留出集为空或全过，且没有安全失败。 */
function gateClean(m: HarborMetrics): boolean {
  const inOk = m.heldInTotal > 0 && m.heldInPass === m.heldInTotal;
  const outOk = m.heldOutTotal === 0 || m.heldOutPass === m.heldOutTotal;
  return inOk && outOk && m.safetyFail === 0;
}

/** 按键名顺序取出第一个非空字符串。 */
function firstHash(raw: Record<string, unknown>, keys: string[]): string {
  for (const k of keys) {
    const v = str(raw[k]);
    if (v) return v;
  }
  return "";
}

/** MaterialChange 一处材料变更。l3 取载荷里的同名布尔。 */
export type MaterialChange = {
  surface: string;
  op: string;
  id: string;
  from: string;
  to: string;
  detail: string;
  l3: boolean;
};

/** LineageNode 谱系上的一个节点：哈希、父节点、变更和是否来自归档。 */
export type LineageNode = {
  hash: string;
  parent: string;
  id: string;
  note: string;
  model: string;
  createdAt: string;
  refs: string[];
  changes: MaterialChange[];
  path: string;
  seed: boolean;
  proposalId: string;
  accepted: boolean;
  fromArchive: boolean;
};

/** HarnessDirs harness 的 home、cas、refs 和 archive 路径。 */
export type HarnessDirs = {
  home: string;
  cas: string;
  refs: string;
  archive: string;
};

/** parseMaterialChange 从松散形状读出一处变更。缺的字段成空串或假。 */
export function parseMaterialChange(v: unknown): MaterialChange {
  return {
    surface: str(pick(v, "surface", "Surface")),
    op: str(pick(v, "op", "Op")),
    id: str(pick(v, "id", "ID")),
    from: str(pick(v, "from", "From")),
    to: str(pick(v, "to", "To")),
    detail: str(pick(v, "detail", "Detail")),
    l3: asBool(pick(v, "l3", "L3")),
  };
}

/** parseMaterialChanges 逐条解析，并丢掉表面、说明和操作都空的项。 */
export function parseMaterialChanges(v: unknown): MaterialChange[] {
  return asArray(v).map(parseMaterialChange).filter((c) => c.surface || c.detail || c.op);
}

/** parseLineageNode 从松散形状读出一个谱系节点。创建时间走 stamp。 */
export function parseLineageNode(v: unknown): LineageNode {
  return {
    hash: str(pick(v, "hash", "Hash")),
    parent: str(pick(v, "parent", "Parent")),
    id: str(pick(v, "id", "ID")),
    note: str(pick(v, "note", "Note")),
    model: str(pick(v, "model", "Model")),
    createdAt: stamp(pick(v, "created_at", "CreatedAt")),
    refs: asArray(pick(v, "refs", "Refs")).map(String).filter(Boolean),
    changes: parseMaterialChanges(pick(v, "changes", "Changes")),
    path: str(pick(v, "path", "Path")),
    seed: asBool(pick(v, "seed", "Seed")),
    proposalId: str(pick(v, "proposal_id", "ProposalID", "proposalId")),
    accepted: asBool(pick(v, "accepted", "Accepted")),
    fromArchive: asBool(pick(v, "from_archive", "FromArchive", "fromArchive")),
  };
}

/** parseLineage 读 lineage，没有则读 nodes。丢掉没有哈希的节点。 */
export function parseLineage(harness: unknown): LineageNode[] {
  const raw = pick(harness, "lineage", "Lineage") ?? pick(harness, "nodes", "Nodes");
  return asArray(raw).map(parseLineageNode).filter((n) => n.hash);
}

/** parseHarnessDirs 从 dirs 里读出四条路径。 */
export function parseHarnessDirs(harness: unknown): HarnessDirs {
  const raw = pick(harness, "dirs", "Dirs") || {};
  return {
    home: str(pick(raw, "home", "Home")),
    cas: str(pick(raw, "cas", "CAS")),
    refs: str(pick(raw, "refs", "Refs")),
    archive: str(pick(raw, "archive", "Archive")),
  };
}

/** formatStamp 把 T 换成空格。长度不少于 16 时截到 16，即到分钟。 */
export function formatStamp(iso: string): string {
  const s = str(iso).replace("T", " ");
  if (s.length >= 16) return s.slice(0, 16);
  return s;
}

/** 字符串原样返回。有限数字当成毫秒时间戳转成 ISO。对象取 Time、time 或 Format。其余收成字符串。空值返回空串。 */
function stamp(v: unknown): string {
  if (v == null) return "";
  if (typeof v === "string") return v;
  if (typeof v === "number" && Number.isFinite(v)) return new Date(v).toISOString();
  if (typeof v === "object") {
    const o = v as Record<string, unknown>;
    return str(o.Time ?? o.time ?? o.Format);
  }
  return str(v);
}
