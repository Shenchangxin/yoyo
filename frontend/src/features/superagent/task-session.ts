/**
 * 拥有：任务页怎么把工人日志和工单输入排成一条会话。
 * 不拥有：拉日志和发送。那一段在 TaskPage。
 */

import type { Item } from "../../lib/protocol";
import type { Assignment, Input, ReportLine, Ticket } from "./desk-model";

export type TaskLine = { id: string; role: string; text: string; name?: string; source?: string; error?: boolean };

const HOST_RESTART = "宿主重启了。先看工作目录、git 和 GitHub，不要重复已经做过的外部动作。";

function hiddenNote(text: string): boolean {
  return text.trim() === HOST_RESTART;
}

/** taskSessionItems 按日志顺序排。派工只露出这一次新写上的话。已经出现过的同一句不再从后来的派工快照里冒出来。只有还没写进日志、又比已经对上的输入更晚的说明，才补在末尾。 */
export function taskSessionItems(ticket: Ticket, lines: TaskLine[]): Item[] {
  const inputs = (ticket.inputs ?? []).filter((row) => row.text?.trim());
  let cursor = 0;
  let prev = "";
  let placed = -1;
  const shown = new Set<string>();
  const said = new Set<string>();
  const out: Item[] = [];
  const place = (index: number) => {
    if (index > placed) placed = index;
  };
  const emit = (index: number) => {
    const row = inputs[index];
    if (!row || shown.has(row.id)) return;
    shown.add(row.id);
    place(index);
    if (hiddenNote(row.text)) return;
    const text = row.text.trim();
    said.add(flat(text));
    out.push(userItem(ticket.id, row.id, text));
  };
  for (const line of lines) {
    if (line.role === "user" && line.source === "assignment") {
      const body = (line.text || "").trim();
      const delta = assignmentDelta(prev, body);
      const spoken = delta || (flat(body) !== flat(prev) ? body : "");
      prev = body || prev;
      const before = cursor;
      cursor = peel(spoken, inputs, cursor, emit);
      if (cursor === before && spoken.trim() && !said.has(flat(spoken))) {
        said.add(flat(spoken));
        out.push(userItem(ticket.id, `log:${line.id}`, spoken.trim()));
        coverInputs(spoken, inputs, shown, place);
      }
      continue;
    }
    if (line.role === "user") {
      const text = (line.text || "").trim();
      if (hiddenNote(text)) continue;
      cursor = nextVisible(inputs, cursor);
      if (cursor < inputs.length && flat(inputs[cursor].text) === flat(text)) {
        emit(cursor);
        cursor += 1;
        continue;
      }
      out.push(lineItem(ticket.id, line));
      said.add(flat(text));
      for (let i = cursor; i < inputs.length; i++) {
        if (flat(inputs[i].text) === flat(text)) {
          shown.add(inputs[i].id);
          place(i);
        }
      }
      continue;
    }
    out.push(lineItem(ticket.id, line));
  }
  for (let i = placed + 1; i < inputs.length; i++) emit(i);
  return withClosing(out, ticket);
}

/** flat 把空白收成一个空格，换行多少不影响是不是同一段话。 */
function flat(text: string): string {
  return text.replace(/\s+/g, " ").trim();
}

/** coverInputs 记下和这段话已经是同一段的输入，避免记录末尾再贴一次。 */
function coverInputs(spoken: string, inputs: Input[], shown: Set<string>, place: (index: number) => void) {
  const said = flat(spoken);
  if (said.length < 40) return;
  inputs.forEach((row, index) => {
    const text = flat(row.text);
    if (!text) return;
    const n = sharedPrefix(said, text);
    if (n >= 40 && n >= Math.min(said.length, text.length) * 0.8) {
      shown.add(row.id);
      place(index);
    }
  });
}

function sharedPrefix(a: string, b: string): number {
  const left = [...a];
  const right = [...b];
  let n = 0;
  while (n < left.length && n < right.length && left[n] === right[n]) n += 1;
  return n;
}

function withClosing(items: Item[], ticket: Ticket): Item[] {
  const seen = items.map((item) => (item.text || "").trim()).filter(Boolean);
  const extra = reportItems(ticket).filter((item) => {
    const text = (item.text || "").trim();
    if (!text || seen.includes(text)) return false;
    return !seen.some((have) => have.includes(text));
  });
  return extra.length ? [...items, ...extra] : items;
}

/** assignmentDelta 只留这一次派工比上一次多出来的正文。对不上前缀时不把整段历史再发一遍。 */
function assignmentDelta(prev: string, body: string): string {
  if (!body) return "";
  if (!prev) return body;
  if (body === prev) return "";
  if (body.startsWith(`${prev}\n`) || body.startsWith(prev)) {
    return body.slice(prev.length).replace(/^\n/, "").trim();
  }
  return "";
}

/** peel 从 delta 开头按整段正文取下一条输入。空白多少不计入。必须顶格，避免「新建吧」撞上「找不到的话就新建吧」。 */
function nextVisible(inputs: { text: string }[], cursor: number): number {
  while (cursor < inputs.length && hiddenNote(inputs[cursor].text)) cursor += 1;
  return cursor;
}

function peel(delta: string, inputs: { text: string }[], cursor: number, emit: (index: number) => void): number {
  let rest = flat(delta);
  while (cursor < inputs.length && rest) {
    cursor = nextVisible(inputs, cursor);
    if (cursor >= inputs.length) break;
    const text = flat(inputs[cursor].text);
    if (!text) {
      cursor += 1;
      continue;
    }
    if (rest === text || rest.startsWith(`${text} `)) {
      emit(cursor);
      rest = rest.slice(text.length).trim();
      cursor += 1;
      continue;
    }
    break;
  }
  return cursor;
}

function userItem(sessionId: string, id: string, text: string): Item {
  return {
    key: id,
    type: "user",
    sessionId,
    source: "user",
    ts: "",
    text,
    name: "",
    delta: false,
    payload: { id },
  };
}

const WROTE = [
  /^wrote \d+ citations to (.+)$/,
  /^replaced \d+ occurrence\(s\) in (.+)$/,
  /^appended \d+ bytes to (.+?) \(file is now \d+ bytes\)$/,
  /^moved .+ -> (.+)$/,
  /^(?:wrote|edited|updated|downloaded) (.+)$/,
];

/** taskFiles 是工人在这张工单里改过的文件，最近改的在前。路径照工具回报的写：相对工作目录，或绝对路径。没写成的和命令行输出都不算。 */
export function taskFiles(lines: TaskLine[]): string[] {
  const out: string[] = [];
  for (const line of lines) {
    if (line.role !== "tool_result" || line.error || /shell|bash|command|terminal/i.test(line.name || "")) continue;
    for (const row of (line.text || "").split("\n")) {
      const path = writtenPath(row.trim());
      if (!path) continue;
      const at = out.indexOf(path);
      if (at >= 0) out.splice(at, 1);
      out.unshift(path);
    }
  }
  return out;
}

function writtenPath(row: string): string {
  for (const pattern of WROTE) {
    const hit = row.match(pattern);
    if (hit) return hit[1].trim();
  }
  return "";
}

function toolPath(name: string, text: string): string {
  const wrote = text.match(/^wrote\s+(\S+)/i);
  if (wrote && (!name || name === "write_file" || name === "create_file" || name === "str_replace" || name === "edit_file")) return wrote[1];
  const hit = text.match(/"path"\s*:\s*"((?:\\.|[^"\\])*)"/);
  if (!hit) return "";
  return hit[1].replace(/\\"/g, "\"").replace(/\\\\/g, "\\");
}

function lineItem(sessionId: string, line: TaskLine): Item {
  const role = line.role === "user" || line.role === "tool_call" || line.role === "tool_result" ? line.role : "assistant";
  const path = toolPath(line.name || "", line.text || "");
  return {
    key: line.role === "tool_result" ? `${line.id}:result` : line.id,
    type: role,
    sessionId,
    source: role === "user" ? "user" : "worker",
    ts: "",
    text: line.text || "",
    name: line.name || "",
    delta: false,
    payload: {
      id: line.id,
      name: line.name || "",
      ...(role === "tool_call" ? { arguments: line.text || "" } : {}),
      ...(role === "tool_result" && line.error ? { ok: false } : {}),
      ...(path ? { path } : {}),
    },
  };
}

function reportItems(ticket: Ticket): Item[] {
  const steps = ticket.assignments ?? [];
  let step: Assignment | undefined;
  for (let i = steps.length - 1; i >= 0; i--) {
    if (reportLines(steps[i]).some(closingLine)) {
      step = steps[i];
      break;
    }
  }
  if (!step) return [];
  const lines = reportLines(step);
  let index = -1;
  for (let i = lines.length - 1; i >= 0; i--) {
    if (closingLine(lines[i])) {
      index = i;
      break;
    }
  }
  if (index < 0) return [];
  const line = lines[index];
  const text = [line.summary, line.question].map((part) => part?.trim() || "").filter((part) => part && part !== "pending" && part !== "running").join("\n\n");
  if (!text) return [];
  return [{
    key: `${step.id}:${index}`,
    type: "assistant",
    sessionId: ticket.id,
    source: "worker",
    ts: "",
    text,
    name: "",
    delta: false,
    payload: { id: `${step.id}:${index}` },
  }];
}

function closingLine(line: ReportLine): boolean {
  return line.state !== "progress" && line.state !== "waiting" && line.state !== "pending" && line.state !== "running";
}

function reportLines(step: Assignment): ReportLine[] {
  if (step.lines && step.lines.length > 0) return step.lines;
  if (step.summary || step.report) return [{ state: step.report || step.state, summary: step.summary }];
  return [];
}
