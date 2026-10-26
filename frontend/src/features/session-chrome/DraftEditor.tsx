/**
 * 拥有：会话草稿的 TipTap 编辑面、占位和提及芯片。Lexical 换成已经带上的 TipTap。
 * 不拥有：发送。只改本地草稿，不调用 api。
 */
import { forwardRef, useEffect, useImperativeHandle, useRef, useState } from "react";
import { EditorContent, useEditor } from "@tiptap/react";
import StarterKit from "@tiptap/starter-kit";
import Placeholder from "@tiptap/extension-placeholder";
import type { Editor } from "@tiptap/core";
import { NodeSelection } from "@tiptap/pm/state";
import { MentionChip } from "./mention-chip";
import { activeAtTail, pinFromToken, type MentionPin } from "../../lib/mentions";
import { cn } from "../../lib/utils";

/** DraftSnapshot 是某一刻的纯文本、芯片、光标和光标前文本。 */
export type DraftSnapshot = { text: string; chips: MentionPin[]; caret: number; before: string };

/** DraftEditorHandle 是草稿编辑器交给外面的插入、替换和快照。 */
export type DraftEditorHandle = {
  insertChip: (pin: MentionPin) => void;
  insertText: (text: string) => void;
  setPlain: (text: string) => void;
  focus: () => void;
  snapshot: () => DraftSnapshot;
  replaceMention: (pin: MentionPin) => void;
  replaceTrigger: (text: string) => void;
  replaceSlash: () => void;
};

/** 编辑器还在、且没被销毁，才算能用。editor 读取 isDestroyed，为假时走另一支。 */
function liveEditor(editor: Editor | null): editor is Editor {
  return !!editor && !editor.isDestroyed;
}

/** 从文档里读出纯文本、芯片和光标。editor 读取 state，传给 liveEditor，传给 textBeforeCaret。 */
function snapshotOf(editor: Editor | null): DraftSnapshot {
  if (!liveEditor(editor)) return { text: "", chips: [], caret: 0, before: "" };
  const chips: MentionPin[] = [];
  const parts: string[] = [];
  editor.state.doc.forEach((block, _offset, index) => {
    if (index > 0) parts.push("\n");
    block.forEach((node) => {
      if (node.type.name === "mentionChip") {
        const token = String(node.attrs.token || "");
        const pin = pinFromToken(token);
        const label = String(node.attrs.label || pin.label);
        chips.push({ ...pin, label });
        return;
      }
      if (node.isText) parts.push(node.text || "");
      if (node.type.name === "hardBreak") parts.push("\n");
    });
  });
  const caret = editor.state.selection.from;
  return { text: parts.join(""), chips, caret, before: textBeforeCaret(editor) };
}

/** 把纯文本收成一段一段的文档。text 按 
 切开后，做映射。 */
function plainToContent(text: string) {
  const lines = text.split("\n");
  return {
    type: "doc" as const,
    content: lines.map((line) => ({
      type: "paragraph" as const,
      content: line ? [{ type: "text" as const, text: line }] : [],
    })),
  };
}

/** 读出光标前的文本。editor 读取 state。 */
function textBeforeCaret(editor: Editor): string {
  const { from } = editor.state.selection;
  return editor.state.doc.textBetween(0, from, "\n", " ");
}

/** DraftEditor 编辑草稿，并把快照交给外面。props 读取 placeholder、onSnapshot、onCaret、onPasteFiles、draftKey、onKeyDown 等，传给 editorAttrs。ref 传给 useImperativeHandle。 */
export const DraftEditor = forwardRef<DraftEditorHandle, {
  draftKey: string;
  disabled?: boolean;
  placeholder: string;
  ariaLabel: string;
  onSnapshot: (snap: DraftSnapshot) => void;
  onCaret?: (snap: DraftSnapshot) => void;
  onKeyDown?: (e: KeyboardEvent, snap: DraftSnapshot, before: string) => boolean;
  onPasteFiles?: (files: File[]) => void;
  testId?: string;
  workspaceTrigger?: boolean;
  pickerOpen?: boolean;
}>(function DraftEditor(props, ref) {
  const placeholderRef = useRef(props.placeholder);
  placeholderRef.current = props.placeholder;
  const onSnapshotRef = useRef(props.onSnapshot);
  onSnapshotRef.current = props.onSnapshot;
  const onCaretRef = useRef(props.onCaret);
  onCaretRef.current = props.onCaret;
  const onPasteRef = useRef(props.onPasteFiles);
  onPasteRef.current = props.onPasteFiles;
  const skipUpdate = useRef(false);
  const editorHolder = useRef<Editor | null>(null);
  const pendingPlain = useRef<string | null>(null);
  const draftKeyRef = useRef(props.draftKey);
  const onKeyDownRef = useRef(props.onKeyDown);
  onKeyDownRef.current = props.onKeyDown;
  const [empty, setEmpty] = useState(true);
  const editor = useEditor({
    // React 19 加上嵌套的 Suspense（映策岛的登录和路由）会拆掉视图，之后才把被动效果接回来。默认同步的第一次渲染会留下过期编辑器，视图是空的时候读取 commands 会抛错。
    immediatelyRender: false,
    extensions: [
      StarterKit.configure({
        heading: false,
        blockquote: false,
        codeBlock: false,
        horizontalRule: false,
        bulletList: false,
        orderedList: false,
        listItem: false,
        bold: false,
        italic: false,
        strike: false,
        code: false,
      }),
      Placeholder.configure({
        /** 占位文字跟当前引用走。 */
        placeholder: () => placeholderRef.current,
        showOnlyCurrent: false,
      }),
      MentionChip,
    ],
    editable: !props.disabled,
    editorProps: {
      attributes: editorAttrs(props),
      /** 剪贴板里有文件就交给粘贴回调，并拦住默认粘贴。_view 函数体内没有读取。event 传给 filesFromClipboard。 */
      handlePaste: (_view, event) => {
        const files = filesFromClipboard(event);
        if (files.length && onPasteRef.current) {
          onPasteRef.current(files);
          return true;
        }
        return false;
      },
      /** 先交给外面的按键回调，再处理芯片上的选择键。_view 函数体内没有读取。event 传给 onKeyDownRef.current，传给 handleChipSelection。 */
      handleKeyDown: (_view, event) => {
        const ed = editorHolder.current;
        if (!liveEditor(ed)) return false;
        if (onKeyDownRef.current?.(event, snapshotOf(ed), textBeforeCaret(ed))) return true;
        return handleChipSelection(ed, event);
      },
    },
    /** 编辑器建好后记下实例，并同步是不是空的。ed 读取 isEmpty，赋给 editorHolder.current。 */
    onCreate: ({ editor: ed }) => {
      editorHolder.current = ed;
      setEmpty(ed.isEmpty);
    },
    /** 编辑器拆掉后清掉实例。 */
    onDestroy: () => {
      editorHolder.current = null;
    },
    /** 正文变了就更新空态，并在不是内部改写时交回快照。ed 读取 isEmpty，传给 snapshotOf。 */
    onUpdate: ({ editor: ed }) => {
      setEmpty(ed.isEmpty);
      if (skipUpdate.current) return;
      onSnapshotRef.current(snapshotOf(ed));
    },
    /** 选区变了，且不是内部改写时，交回光标位置。ed 传给 snapshotOf。 */
    onSelectionUpdate: ({ editor: ed }) => {
      if (skipUpdate.current) return;
      onCaretRef.current?.(snapshotOf(ed));
    },
  });
  editorHolder.current = liveEditor(editor) ? editor : null;

  useEffect(() => {
    const keyChanged = draftKeyRef.current !== props.draftKey;
    draftKeyRef.current = props.draftKey;
    if (!liveEditor(editor)) {
      if (keyChanged) pendingPlain.current = null;
      return;
    }
    if (keyChanged) {
      pendingPlain.current = null;
      skipUpdate.current = true;
      editor.commands.clearContent();
      skipUpdate.current = false;
      setEmpty(true);
      return;
    }
    if (pendingPlain.current === null) return;
    const text = pendingPlain.current;
    pendingPlain.current = null;
    editor.commands.setContent(plainToContent(text));
    setEmpty(!text.trim());
    if (text.trim()) editor.commands.focus("end");
  }, [props.draftKey, editor]);

  useEffect(() => {
    if (!liveEditor(editor)) return;
    editor.setEditable(!props.disabled);
  }, [editor, props.disabled]);

  useEffect(() => {
    if (!liveEditor(editor)) return;
    editor.setOptions({
      editorProps: {
        ...editor.options.editorProps,
        attributes: editorAttrs(props),
      },
    });
  }, [editor, props.ariaLabel, props.testId, props.workspaceTrigger, props.pickerOpen]);

  useImperativeHandle(ref, () => ({
    /** 插入一枚提及芯片。pin 传给 insertMentionChip。 */
    insertChip: (pin) => {
      if (!liveEditor(editor)) return;
      editor.chain().focus().insertMentionChip(pin).run();
    },
    /** 在光标处插入文本。text 传给 insertContent。 */
    insertText: (text) => {
      if (!liveEditor(editor)) return;
      editor.chain().focus().insertContent(text).run();
    },
    /** 用纯文本换掉全文；编辑器还没好就先记下。text 去掉首尾空白，为假时走另一支，赋给 pendingPlain.current，传给 plainToContent。 */
    setPlain: (text) => {
      if (!liveEditor(editor)) {
        pendingPlain.current = text;
        return;
      }
      pendingPlain.current = null;
      skipUpdate.current = true;
      editor.commands.setContent(plainToContent(text));
      skipUpdate.current = false;
      setEmpty(!text.trim());
    },
    /** 把光标放到末尾。 */
    focus: () => {
      if (!liveEditor(editor)) return;
      editor.commands.focus("end");
    },
    /** 交回当前快照；编辑器还没好就用挂起的纯文本。 */
    snapshot: () => {
      if (liveEditor(editor)) return snapshotOf(editor);
      const text = pendingPlain.current ?? "";
      return { text, chips: [], caret: text.length, before: text };
    },
    /** 把光标前的提及换成芯片，没有就直接插入。pin 传给 insertMentionChip。 */
    replaceMention: (pin) => {
      if (!liveEditor(editor)) return;
      const token = activeAtTail(textBeforeCaret(editor));
      if (!token) {
        editor.chain().focus().insertMentionChip(pin).run();
        return;
      }
      const from = editor.state.selection.from - token.length;
      const to = editor.state.selection.from;
      editor.chain().focus().deleteRange({ from, to }).insertMentionChip(pin).run();
    },
    /** 把光标前的触发词换成文本，没有就直接插入。text 传给 insertContent。 */
    replaceTrigger: (text) => {
      if (!liveEditor(editor)) return;
      const token = activeAtTail(textBeforeCaret(editor));
      if (!token) {
        editor.chain().focus().insertContent(text).run();
        return;
      }
      const from = editor.state.selection.from - token.length;
      const to = editor.state.selection.from;
      editor.chain().focus().deleteRange({ from, to }).insertContent(text).run();
    },
    /** 删掉光标前的斜线命令。 */
    replaceSlash: () => {
      if (!liveEditor(editor)) return;
      const before = textBeforeCaret(editor);
      const m = /(?:^|\s)(\/[^\s]*)$/.exec(before);
      if (!m?.[1]) return;
      const token = m[1];
      const from = editor.state.selection.from - token.length;
      const to = editor.state.selection.from;
      editor.chain().focus().deleteRange({ from, to }).run();
    },
  }), [editor]);

  return (
    <div className={cn("sc-scroll")} data-input-scroll>
      <div className="sc-grow">
        {empty ? (
          <div className="sc-placeholder" aria-hidden="true">
            {props.placeholder}
          </div>
        ) : null}
        <EditorContent editor={editor} />
      </div>
    </div>
  );
});

/** 拼出输入框的无障碍属性和测试标记。props 读取 ariaLabel、testId、workspaceTrigger、pickerOpen。 */
function editorAttrs(props: {
  ariaLabel: string;
  testId?: string;
  workspaceTrigger?: boolean;
  pickerOpen?: boolean;
}): Record<string, string> {
  const attrs: Record<string, string> = {
    class: "sc-input",
    "aria-label": props.ariaLabel,
    "data-testid": props.testId || "composer-input",
  };
  if (props.workspaceTrigger) {
    attrs["aria-haspopup"] = "dialog";
    attrs["aria-expanded"] = props.pickerOpen ? "true" : "false";
  }
  return attrs;
}

const PASSIVE_KEYS = new Set(["Shift", "Control", "Alt", "Meta", "Unidentified", "Process"]);

/** 退格或删除先选中旁边的芯片，再按一次才删掉。editor 读取 state、view，调用 chain，传给 clearChipSelected。event 读取 key。 */
function handleChipSelection(editor: Editor, event: KeyboardEvent): boolean {
  if (event.key !== "Backspace" && event.key !== "Delete") {
    if (!PASSIVE_KEYS.has(event.key) && event.key.length) clearChipSelected(editor);
    return false;
  }
  const sel = editor.state.selection;
  if (sel instanceof NodeSelection && sel.node.type.name === "mentionChip") return false;
  const backward = event.key === "Backspace";
  const $from = sel.$from;
  if (!sel.empty) return false;
  const node = backward ? $from.nodeBefore : $from.nodeAfter;
  if (!node || node.type.name !== "mentionChip") {
    clearChipSelected(editor);
    return false;
  }
  const pos = backward ? $from.pos - node.nodeSize : $from.pos;
  const dom = editor.view.nodeDOM(pos);
  const el = dom instanceof HTMLElement ? (dom.closest("[data-mention-chip]") || (dom.hasAttribute("data-mention-chip") ? dom : null)) : null;
  if (!(el instanceof HTMLElement)) return false;
  if (el.hasAttribute("data-selected")) {
    clearChipSelected(editor);
    editor.chain().focus().deleteRange({ from: pos, to: pos + node.nodeSize }).run();
    return true;
  }
  clearChipSelected(editor);
  el.setAttribute("data-selected", "");
  editor.view.dom.setAttribute("data-composer-chip-selected", "");
  return true;
}

/** 清掉芯片的选中标记。editor 读取 view。 */
function clearChipSelected(editor: Editor) {
  editor.view.dom.removeAttribute("data-composer-chip-selected");
  editor.view.dom.querySelectorAll("[data-mention-chip][data-selected]").forEach((n) => n.removeAttribute("data-selected"));
}

/** 从剪贴板里取出文件。e 读取 clipboardData。 */
function filesFromClipboard(e: ClipboardEvent): File[] {
  const files: File[] = [];
  const items = e.clipboardData?.items;
  if (items) {
    for (let i = 0; i < items.length; i++) {
      const item = items[i];
      if (item.kind === "file" || item.type.startsWith("image/")) {
        const f = item.getAsFile();
        if (f) files.push(f);
      }
    }
  }
  return files;
}
