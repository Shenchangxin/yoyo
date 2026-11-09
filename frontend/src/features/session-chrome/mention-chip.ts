/**
 * 拥有：行内提及芯片。用 TipTap 原子节点，光标几何保持原生。agent 芯片前面多一个字形。
 * 不拥有：草稿正文。不调用 api。
 */
import { Node, mergeAttributes } from "@tiptap/core";
import type { DOMOutputSpec } from "@tiptap/pm/model";
import type { MentionKind, MentionPin } from "../../lib/mentions";

declare module "@tiptap/core" {
  interface Commands<ReturnType> {
    mentionChip: {
      insertMentionChip: (pin: MentionPin) => ReturnType;
    };
  }
}

/** SVG_ICON 是能放进 style 的图标：后端给的 base64 SVG。style 是整串声明，带引号或括号的值会写出别的声明，所以别的一律不认。 */
const SVG_ICON = /^data:image\/svg\+xml;base64,[A-Za-z0-9+/]+=*$/;

/** agentMark 是 agent 芯片前的字形：有图标时当遮罩跟着文字着色，没有时用首字母。字母由样式画出，不进芯片的文字。 */
function agentMark(icon: string, label: string): DOMOutputSpec {
  if (SVG_ICON.test(icon)) {
    const url = `url("${icon}")`;
    return ["span", { class: "sc-mention-mark", "aria-hidden": "true", style: `mask-image:${url};-webkit-mask-image:${url}` }];
  }
  const letter = label.trim().charAt(0).toUpperCase() || "?";
  return ["span", { class: "sc-mention-mark sc-mention-letter", "aria-hidden": "true", "data-letter": letter }];
}

/** MentionChip 是一枚行内提及芯片，光标按原子节点走。 */
export const MentionChip = Node.create({
  name: "mentionChip",
  group: "inline",
  inline: true,
  atom: true,
  selectable: true,
  /** 芯片带上记号、标签、种类，agent 还带图标。 */
  addAttributes() {
    return {
      token: { default: "" },
      label: { default: "" },
      kind: { default: "file" as MentionKind },
      icon: { default: "" },
    };
  },
  /** 从带标记的 span 读回芯片。 */
  parseHTML() {
    return [{ tag: "span[data-mention-chip]" }];
  },
  /** 画成一枚带记号和标签的 span。HTMLAttributes 读取 token、kind、label、icon。 */
  renderHTML({ HTMLAttributes }) {
    const token = String(HTMLAttributes.token || "");
    const label = String(HTMLAttributes.label || token);
    const attrs = mergeAttributes({
      "data-mention-chip": "",
      "data-testid": "mention-chip",
      "data-kind": HTMLAttributes.kind,
      class: "sc-mention",
      "data-token": token,
      "aria-label": token,
    });
    if (HTMLAttributes.kind === "agent") return ["span", attrs, agentMark(String(HTMLAttributes.icon || ""), label), label];
    return ["span", attrs, label];
  },
  /** 提供插入芯片并补一个空格的命令。 */
  addCommands() {
    return {
      /** 在光标处插入芯片，并跟上一个空格。pin 读取 token、label、kind、icon。 */
      insertMentionChip: (pin: MentionPin) => ({ chain }) =>
        chain()
          .insertContent({
            type: this.name,
            attrs: { token: pin.token, label: pin.label, kind: pin.kind, icon: pin.icon || "" },
          })
          .insertContent(" ")
          .run(),
    };
  },
});
