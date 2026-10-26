/**
 * 拥有：行内提及芯片。用 TipTap 原子节点，光标几何保持原生。
 * 不拥有：草稿正文。不调用 api。
 */
import { Node, mergeAttributes } from "@tiptap/core";
import type { MentionKind, MentionPin } from "../../lib/mentions";

declare module "@tiptap/core" {
  interface Commands<ReturnType> {
    mentionChip: {
      insertMentionChip: (pin: MentionPin) => ReturnType;
    };
  }
}

/** MentionChip 是一枚行内提及芯片，光标按原子节点走。 */
export const MentionChip = Node.create({
  name: "mentionChip",
  group: "inline",
  inline: true,
  atom: true,
  selectable: true,
  /** 芯片带上记号、标签和种类。 */
  addAttributes() {
    return {
      token: { default: "" },
      label: { default: "" },
      kind: { default: "file" as MentionKind },
    };
  },
  /** 从带标记的 span 读回芯片。 */
  parseHTML() {
    return [{ tag: "span[data-mention-chip]" }];
  },
  /** 画成一枚带记号和标签的 span。 */
  renderHTML({ HTMLAttributes }) {
    const token = String(HTMLAttributes.token || "");
    return [
      "span",
      mergeAttributes({
        "data-mention-chip": "",
        "data-testid": "mention-chip",
        "data-kind": HTMLAttributes.kind,
        class: "sc-mention",
        "data-token": token,
        "aria-label": token,
      }),
      String(HTMLAttributes.label || token),
    ];
  },
  /** 提供插入芯片并补一个空格的命令。 */
  addCommands() {
    return {
      /** 在光标处插入芯片，并跟上一个空格。 */
      insertMentionChip: (pin: MentionPin) => ({ chain }) =>
        chain()
          .insertContent({
            type: this.name,
            attrs: { token: pin.token, label: pin.label, kind: pin.kind },
          })
          .insertContent(" ")
          .run(),
    };
  },
});
