/**
 * 拥有：默认快捷键，以及按键事件与快捷键字符串的匹配和显示。
 * 不拥有：快捷键实际触发的命令。
 */

/** DEFAULT_KEYMAP 命令面板、新对话、设置、审阅、once、session、always、Escape，以及上下回合。 */
export const DEFAULT_KEYMAP = {
  palette: "Mod+K",
  newChat: "Mod+N",
  control: "Mod+,",
  toggleReview: "Mod+\\",
  once: "1",
  session: "2",
  always: "3",
  deny: "Escape",
  prevTurn: "Alt+ArrowUp",
  nextTurn: "Alt+ArrowDown",
} as const;

/** KeymapId 默认表里的键名。 */
export type KeymapId = keyof typeof DEFAULT_KEYMAP;
/** Keymap 每个键名对应一条快捷键字符串。 */
export type Keymap = Record<KeymapId, string>;

/** mergeKeymap 用传入值盖住默认表。空字符串不覆盖。表外的键名忽略。raw 为假时返回 next，否则返回 next；按下标读取。 */
export function mergeKeymap(raw?: Record<string, string> | null): Keymap {
  const next: Keymap = { ...DEFAULT_KEYMAP };
  if (!raw) return next;
  (Object.keys(DEFAULT_KEYMAP) as KeymapId[]).forEach((id) => {
    const v = (raw[id] || "").trim();
    if (v) next[id] = v;
  });
  return next;
}

/**
 * matchKey 最后一段是键，前面是修饰。Mod、CmdOrCtrl、Ctrl、Meta、Cmd 都看成按下了 Ctrl 或 Meta。Escape、逗号按键值匹配。反斜杠也认 Backslash。
 * e 读取 ctrlKey、metaKey、shiftKey、altKey、key、code。
 * spec 按 + 切开，做映射。
 */
export function matchKey(e: KeyboardEvent, spec: string): boolean {
  const parts = spec.split("+").map((p) => p.trim()).filter(Boolean);
  if (!parts.length) return false;
  const keyName = parts[parts.length - 1];
  const mods = parts.slice(0, -1).map((m) => m.toLowerCase());
  const wantMod = mods.some((m) => m === "mod" || m === "cmdorctrl" || m === "ctrl" || m === "meta" || m === "cmd");
  const wantShift = mods.includes("shift");
  const wantAlt = mods.includes("alt");
  const hasMod = e.ctrlKey || e.metaKey;
  if (wantMod !== hasMod) return false;
  if (wantShift !== e.shiftKey) return false;
  if (wantAlt !== e.altKey) return false;
  const k = e.key;
  if (keyName.toLowerCase() === "escape") return k === "Escape";
  if (keyName === ",") return k === ",";
  if (keyName === "\\") return k === "\\" || e.code === "Backslash";
  return k.toLowerCase() === keyName.toLowerCase();
}

/** displayShortcut Mac 上修饰键收成符号且不再用加号连接，其他平台保留加号。Escape 显示成 Esc，方向键显示成箭头。spec 按 + 切开，做映射。 */
export function displayShortcut(spec: string): string {
  const mac = typeof navigator !== "undefined" && /Mac|iPhone|iPad/i.test(navigator.platform || navigator.userAgent);
  return spec
    .split("+")
    .map((part) => {
      const p = part.trim();
      if (p === "Mod" || p === "CmdOrCtrl") return mac ? "⌘" : "Ctrl";
      if (p === "Shift") return mac ? "⇧" : "Shift";
      if (p === "Alt") return mac ? "⌥" : "Alt";
      if (p === "Escape") return "Esc";
      if (p === "ArrowUp") return "↑";
      if (p === "ArrowDown") return "↓";
      if (p === ",") return ",";
      if (p === "\\") return "\\";
      return p;
    })
    .join(mac ? "" : "+");
}

/** formatShortcut 从按键事件收成快捷键字符串。只按下修饰键时返回空。空格记成 Space。单字符一律记成大写。Shift 在键不是单字符，或单字符仍是小写时记入。e 读取 key、metaKey、ctrlKey、altKey、shiftKey。 */
export function formatShortcut(e: { key: string; metaKey: boolean; ctrlKey: boolean; altKey: boolean; shiftKey: boolean }): string {
  const ignore = new Set(["Control", "Meta", "Shift", "Alt"]);
  if (ignore.has(e.key)) return "";
  const parts: string[] = [];
  if (e.metaKey || e.ctrlKey) parts.push("Mod");
  if (e.altKey) parts.push("Alt");
  if (e.shiftKey && e.key.length !== 1) parts.push("Shift");
  if (e.shiftKey && e.key.length === 1 && e.key.toLowerCase() === e.key) parts.push("Shift");
  let key = e.key;
  if (key === " ") key = "Space";
  if (key === "Escape") key = "Escape";
  if (key.length === 1) key = e.shiftKey ? key.toUpperCase() : key.toUpperCase();
  if (key === "\\") key = "\\";
  parts.push(key);
  return parts.join("+");
}
