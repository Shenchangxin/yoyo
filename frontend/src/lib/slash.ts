export type SlashCommand = {
  cmd: string;
  hint: string;
};

export const SLASH: SlashCommand[] = [
  { cmd: "/new", hint: "New chat" },
  { cmd: "/rename", hint: "Rename thread: /rename title" },
  { cmd: "/fork", hint: "Fork this thread" },
  { cmd: "/archive", hint: "Archive this thread" },
  { cmd: "/delete", hint: "Delete this thread" },
  { cmd: "/plan", hint: "Toggle plan mode" },
  { cmd: "/diff", hint: "Refresh review diff" },
  { cmd: "/steer", hint: "Steer the running turn" },
  { cmd: "/compact", hint: "Compact context" },
  { cmd: "/model", hint: "Set thread model: /model id" },
  { cmd: "/export", hint: "Export thread as markdown" },
  { cmd: "/stop", hint: "Stop the running turn" },
  { cmd: "/quit", hint: "Quit Yoyo" },
];

export function slashQuery(value: string): string | null {
  const last = (value.split("\n").pop() || "");
  if (!last.startsWith("/")) return null;
  if (/\s/.test(last.slice(1))) return null;
  return last.toLowerCase();
}

export function filterSlash(prefix: string): SlashCommand[] {
  return SLASH.filter((s) => s.cmd.startsWith(prefix) || s.cmd.slice(1).startsWith(prefix.slice(1)));
}
