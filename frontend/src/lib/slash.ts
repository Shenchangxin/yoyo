import type { Copy } from "./copy";

export type SlashCommand = {
  cmd: string;
  hint: string;
};

export function slashCatalog(copy: Copy): SlashCommand[] {
  return [
    { cmd: "/new", hint: copy.slash.new },
    { cmd: "/rename", hint: copy.slash.rename },
    { cmd: "/fork", hint: copy.slash.fork },
    { cmd: "/archive", hint: copy.slash.archive },
    { cmd: "/delete", hint: copy.slash.delete },
    { cmd: "/plan", hint: copy.slash.plan },
    { cmd: "/diff", hint: copy.slash.diff },
    { cmd: "/steer", hint: copy.slash.steer },
    { cmd: "/compact", hint: copy.slash.compact },
    { cmd: "/rewind", hint: copy.slash.rewind },
    { cmd: "/context", hint: copy.slash.context },
    { cmd: "/model", hint: copy.slash.model },
    { cmd: "/export", hint: copy.slash.export },
    { cmd: "/stop", hint: copy.slash.stop },
    { cmd: "/quit", hint: copy.slash.quit },
    { cmd: "/schedule", hint: copy.slash.schedule },
    { cmd: "/remember", hint: copy.slash.remember },
    { cmd: "/forget", hint: copy.slash.forget },
    { cmd: "/project", hint: copy.slash.project },
    { cmd: "/artifact", hint: copy.slash.artifact },
    { cmd: "/improve", hint: copy.slash.improve },
  ];
}

/** `/token` at the caret: line start or after whitespace, same as harness. */
export function slashQuery(value: string): string | null {
  const m = /(?:^|\s)(\/[^\s]*)$/.exec(value);
  return m?.[1] ? m[1].toLowerCase() : null;
}

export function filterSlash(prefix: string, items: SlashCommand[]): SlashCommand[] {
  return items.filter((s) => s.cmd.startsWith(prefix) || s.cmd.slice(1).startsWith(prefix.slice(1)));
}
