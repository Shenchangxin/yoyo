import type { Approval } from "./protocol";

/** Idle window before an untouched question card continues with the suggestion. */
export const ASK_IDLE_MS = 10_000;

export type AskOption = { id: string; label: string };
export type AskQuestion = {
  id: string;
  prompt: string;
  options: AskOption[];
  recommended: string;
};
export type AskForm = { title: string; questions: AskQuestion[] };

export function pendingAsk(approvals: Approval[], sessionId?: string): Approval | undefined {
  return approvals.find((a) => a.action === "ask_user" && (!sessionId || !a.sessionId || a.sessionId === sessionId));
}

export function parseAskCommand(command: string): AskForm {
  const text = command.trim();
  if (text.startsWith("{")) {
    try {
      const raw = JSON.parse(text) as { title?: unknown; questions?: unknown };
      const questions = normalizeQuestions(raw.questions);
      if (questions.length) return { title: textOf(raw.title), questions };
    } catch {
      /* a plain question, not a form */
    }
  }
  return { title: "", questions: [{ id: "q1", prompt: text, options: [], recommended: "" }] };
}

/** A card can continue on its own only when every question already has a suggestion. */
export function askCanIdle(form: AskForm): boolean {
  return form.questions.length > 0 && form.questions.every((q) => q.options.length > 0);
}

export function recommendedPicks(form: AskForm): Record<string, string> {
  const picks: Record<string, string> = {};
  for (const q of form.questions) {
    if (q.recommended) picks[q.id] = q.recommended;
    else if (q.options[0]) picks[q.id] = q.options[0].id;
  }
  return picks;
}

export function idleAnswer(form: AskForm): string {
  return formatAsk(form, recommendedPicks(form), {}, true);
}

export function chosenAnswer(form: AskForm, picks: Record<string, string>, free: Record<string, string>): string {
  return formatAsk(form, picks, free, false);
}

function formatAsk(form: AskForm, picks: Record<string, string>, free: Record<string, string>, idle: boolean): string {
  const lines = form.questions.map((q) => {
    if (!q.options.length) {
      const typed = (free[q.id] || "").trim() || "continue with your best judgment";
      return `${q.id}: ${typed}`;
    }
    const id = picks[q.id] || q.recommended || q.options[0].id;
    const opt = q.options.find((o) => o.id === id) || q.options[0];
    return `${q.id}: ${opt.id} — ${opt.label}`;
  });
  lines.push(idle ? "note: operator idle; used recommended" : "note: operator answered");
  return lines.join("\n");
}

function normalizeQuestions(raw: unknown): AskQuestion[] {
  if (!Array.isArray(raw)) return [];
  const out: AskQuestion[] = [];
  for (const item of raw) {
    if (!item || typeof item !== "object") continue;
    const rec = item as Record<string, unknown>;
    const prompt = textOf(rec.prompt || rec.question);
    if (!prompt) continue;
    const options = normalizeOptions(rec.options).slice(0, 8);
    let recommended = textOf(rec.recommended);
    if (options.length && !options.some((o) => o.id === recommended)) recommended = options[0].id;
    if (!options.length) recommended = "";
    out.push({
      id: textOf(rec.id) || `q${out.length + 1}`,
      prompt,
      options,
      recommended,
    });
    if (out.length >= 6) break;
  }
  return out;
}

function normalizeOptions(raw: unknown): AskOption[] {
  if (!Array.isArray(raw)) return [];
  const out: AskOption[] = [];
  for (const item of raw) {
    if (typeof item === "string") {
      const label = item.trim();
      if (label) out.push({ id: String(out.length + 1), label });
      continue;
    }
    if (!item || typeof item !== "object") continue;
    const rec = item as Record<string, unknown>;
    const label = textOf(rec.label) || textOf(rec.id);
    if (!label) continue;
    out.push({ id: textOf(rec.id) || String(out.length + 1), label });
  }
  return out;
}

function textOf(v: unknown): string {
  return String(v ?? "").trim();
}
