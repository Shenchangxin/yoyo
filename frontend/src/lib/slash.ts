/**
 * 拥有：从光标处识别斜杠命令，以及按前缀过滤命令目录。
 * 不拥有：命令的执行，以及提及菜单。
 */

import type { Copy } from "./copy";

/** SlashCommand 一条斜杠命令及其提示。 */
export type SlashCommand = {
  cmd: string;
  hint: string;
};

/** slashCatalog 按文案表拼出可补全的命令目录。copy 读取 slash。 */
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

/** slashQuery 光标处的 `/token`：行首或空白之后，与 harness 相同。没有则返回空。结果为小写。value 传给 exec。 */
export function slashQuery(value: string): string | null {
  const m = /(?:^|\s)(\/[^\s]*)$/.exec(value);
  return m?.[1] ? m[1].toLowerCase() : null;
}

/** filterSlash 留下命令或去掉斜杠后的名字以前缀开头的项。prefix 截取，传给 startsWith。items 做过滤。 */
export function filterSlash(prefix: string, items: SlashCommand[]): SlashCommand[] {
  return items.filter((s) => s.cmd.startsWith(prefix) || s.cmd.slice(1).startsWith(prefix.slice(1)));
}
