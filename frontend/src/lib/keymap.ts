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
