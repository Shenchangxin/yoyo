export const DEFAULT_KEYMAP = {
  palette: "Mod+K",
  newChat: "Mod+N",
  control: "Mod+,",
  toggleReview: "Mod+\\",
  once: "1",
  session: "2",
  always: "3",
  deny: "Escape",
} as const;

export type KeymapId = keyof typeof DEFAULT_KEYMAP;
export type Keymap = Record<KeymapId, string>;

export function mergeKeymap(raw?: Record<string, string> | null): Keymap {
  const next: Keymap = { ...DEFAULT_KEYMAP };
  if (!raw) return next;
  (Object.keys(DEFAULT_KEYMAP) as KeymapId[]).forEach((id) => {
    const v = (raw[id] || "").trim();
    if (v) next[id] = v;
  });
  return next;
}

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
      if (p === ",") return ",";
      if (p === "\\") return "\\";
      return p;
    })
    .join(mac ? "" : "+");
}

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
