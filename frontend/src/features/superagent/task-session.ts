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

/** taskSessionItems 按日志顺序排。派工只露出这一次新写上的话，短句不会借旧句子里的同样几个字提前出现。 */
export function taskSessionItems(ticket: Ticket, lines: TaskLine[]): Item[] {
  const inputs = (ticket.inputs ?? []).filter((row) => row.text?.trim());
  let cursor = 0;
  let prev = "";
  const shown = new Set<string>();
  const out: Item[] = [];
  const emit = (row: Input) => {
    if (shown.has(row.id)) return;
    shown.add(row.id);
    if (hiddenNote(row.text)) return;
    out.push(userItem(ticket.id, row.id, row.text.trim()));
  };
  for (const line of lines) {
    if (line.role === "user" && line.source === "assignment") {
      const body = (line.text || "").trim();
      const delta = assignmentDelta(prev, body);
      prev = body || prev;
      cursor = peel(delta, inputs, cursor, emit);
      continue;
    }
    if (line.role === "user") {
      const text = (line.text || "").trim();
      if (hiddenNote(text)) continue;
      if (cursor < inputs.length && inputs[cursor].text.trim() === text) {
        emit(inputs[cursor]);
        cursor += 1;
        continue;
      }
      out.push(lineItem(ticket.id, line));
      continue;
    }
    out.push(lineItem(ticket.id, line));
  }
  while (cursor < inputs.length) {
    const row = inputs[cursor];
    cursor += 1;
    if (!shown.has(row.id) && row.text.trim()) emit(row);
  }
  return withClosing(out, ticket);
}

function withClosing(items: Item[], ticket: Ticket): Item[] {
  const seen = new Set(items.map((item) => (item.text || "").trim()).filter(Boolean));
  const extra = reportItems(ticket).filter((item) => {
    const text = (item.text || "").trim();
    return text && !seen.has(text);
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

/** peel 从 delta 开头按整段正文取下一条输入。必须顶格，避免「新建吧」撞上「找不到的话就新建吧」。 */
function peel(delta: string, inputs: Input[], cursor: number, emit: (row: Input) => void): number {
  let rest = delta.trim();
  while (cursor < inputs.length && rest) {
    const text = inputs[cursor].text.trim();
    if (!text) {
      cursor += 1;
      continue;
    }
    if (rest === text || rest.startsWith(`${text}\n`)) {
      emit(inputs[cursor]);
      rest = rest.slice(text.length).replace(/^\n/, "").trim();
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
  const items: Item[] = [];
  for (const step of ticket.assignments ?? []) {
    reportLines(step).forEach((line, index) => {
      if (line.state === "progress" || line.state === "waiting" || line.state === "pending" || line.state === "running") return;
      const text = [line.summary, line.question].map((part) => part?.trim() || "").filter((part) => part && part !== "pending" && part !== "running").join("\n\n");
      if (!text) return;
      items.push({
        key: `${step.id}:${index}`,
        type: "assistant",
        sessionId: ticket.id,
        source: "worker",
        ts: "",
        text,
        name: "",
        delta: false,
        payload: { id: `${step.id}:${index}` },
      });
    });
  }
  return items;
}

function reportLines(step: Assignment): ReportLine[] {
  if (step.lines && step.lines.length > 0) return step.lines;
  if (step.summary || step.report) return [{ state: step.report || step.state, summary: step.summary }];
  return [];
}
