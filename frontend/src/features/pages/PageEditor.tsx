/** 拥有一页的标题、正文工具条和保存状态。不拥有页面树。正文改动交给回调，由外面保存到服务端。 */
import { useEffect, useRef, useState, type ReactNode } from "react";
import Placeholder from "@tiptap/extension-placeholder";
import { EditorContent, useEditor, type Editor } from "@tiptap/react";
import StarterKit from "@tiptap/starter-kit";
import { Bold, Code2, Heading2, Italic, List, ListOrdered, MessageSquare, MoreHorizontal, Quote, Trash2, Unlink } from "lucide-react";
import { getLocale, useCopy } from "../../lib/i18n";
import { Button } from "../../components/ui/button";
import { Input } from "../../components/ui/input";
import { Tooltip } from "../../components/ui/tooltip";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "../../components/ui/dropdown-menu";
import { cn } from "../../lib/utils";
import { htmlToMd, mdToHtml } from "./markdown";
import { formatPageWhen, pageMeasure } from "./when";

/** SaveState 是已存、正在存、已改、冲突或出错。 */
export type SaveState = "saved" | "saving" | "dirty" | "conflict" | "error";

/**
 * PageEditor 编辑标题和正文，保存与删除交给外面。
 * props 读取 title、content、onSave、state、updatedAt、bound 等，调用 onContent，调用 onSave，调用 onNewChild，调用 onMoveTop，调用 onReload，调用 onDelete，调用 onTitle。
 */
export function PageEditor(props: {
  title: string;
  content: string;
  revision: number;
  state: SaveState;
  error?: string;
  bound?: boolean;
  parentId?: string;
  updatedAt?: string;
  onTitle: (t: string) => void;
  onContent: (md: string) => void;
  onSave: () => void;
  onReload?: () => void;
  onBind?: () => void;
  onUnbind?: () => void;
  onDelete?: () => void;
  onNewChild?: () => void;
  onMoveTop?: () => void;
}) {
  const copy = useCopy();
  const skip = useRef(false);
  const [localTitle, setLocalTitle] = useState(props.title);
  const editor = useEditor({
    immediatelyRender: false,
    extensions: [
      StarterKit.configure({ heading: { levels: [1, 2, 3] } }),
      Placeholder.configure({ placeholder: copy.pages.bodyPh }),
    ],
    content: mdToHtml(props.content),
    editorProps: { attributes: { class: "page-editor min-h-[52vh] w-full max-w-3xl px-1 py-1 text-[16px] leading-7 focus:outline-none" } },
    /** 不是内部回写时，把 HTML 收成 Markdown 交给外面。ed 调用 getHTML。 */
    onUpdate: ({ editor: ed }) => {
      if (skip.current) return;
      props.onContent(htmlToMd(ed.getHTML()));
    },
  });

  useEffect(() => {
    setLocalTitle(props.title);
  }, [props.title]);

  useEffect(() => {
    if (!editor) return;
    skip.current = true;
    editor.commands.setContent(mdToHtml(props.content), { emitUpdate: false });
    skip.current = false;
  }, [editor]);

  useEffect(() => {
    /** 用保存快捷键触发外面的保存。e 读取 metaKey、ctrlKey、key，调用 preventDefault。 */
    const onKey = (e: KeyboardEvent) => {
      if ((e.metaKey || e.ctrlKey) && e.key.toLowerCase() === "s") {
        e.preventDefault();
        props.onSave();
      }
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [props.onSave]);

  useEffect(() => {
    const dirty = props.state === "dirty" || props.state === "saving";
    /** 还有未保存的改动时，离开前提醒。e 读取 returnValue，调用 preventDefault。 */
    const leave = (e: BeforeUnloadEvent) => {
      if (!dirty) return;
      e.preventDefault();
      e.returnValue = "";
    };
    window.addEventListener("beforeunload", leave);
    return () => window.removeEventListener("beforeunload", leave);
  }, [props.state]);

  const locale = getLocale();
  const measure = pageMeasure(props.content, locale);
  const measureLabel = (measure.kind === "chars" ? copy.pages.chars : copy.pages.words).replace("{n}", String(measure.n));
  const when = formatPageWhen(props.updatedAt || "", locale);

  return (
    <div className="page-sheet flex min-h-0 flex-1 flex-col" data-testid="page-editor">
      <div className="flex flex-wrap items-center gap-1.5 border-b border-border/50 px-3 py-1.5 sm:px-5">
        <div className="flex min-w-0 items-center gap-0.5">
          <MarkBtn editor={editor} label={copy.pages.heading2} active={editor?.isActive("heading", { level: 2 })} onClick={() => editor?.chain().focus().toggleHeading({ level: 2 }).run()}>
            <Heading2 className="size-3.5" />
          </MarkBtn>
          <MarkBtn editor={editor} label={copy.pages.bold} active={editor?.isActive("bold")} onClick={() => editor?.chain().focus().toggleBold().run()}>
            <Bold className="size-3.5" />
          </MarkBtn>
          <MarkBtn editor={editor} label={copy.pages.italic} active={editor?.isActive("italic")} onClick={() => editor?.chain().focus().toggleItalic().run()}>
            <Italic className="size-3.5" />
          </MarkBtn>
          <MarkBtn editor={editor} label={copy.pages.list} active={editor?.isActive("bulletList")} onClick={() => editor?.chain().focus().toggleBulletList().run()}>
            <List className="size-3.5" />
          </MarkBtn>
          <MarkBtn editor={editor} label={copy.pages.ordered} active={editor?.isActive("orderedList")} onClick={() => editor?.chain().focus().toggleOrderedList().run()}>
            <ListOrdered className="size-3.5" />
          </MarkBtn>
          <MarkBtn editor={editor} label={copy.pages.quote} active={editor?.isActive("blockquote")} onClick={() => editor?.chain().focus().toggleBlockquote().run()}>
            <Quote className="size-3.5" />
          </MarkBtn>
          <MarkBtn editor={editor} label={copy.pages.code} active={editor?.isActive("code")} onClick={() => editor?.chain().focus().toggleCode().run()}>
            <Code2 className="size-3.5" />
          </MarkBtn>
        </div>
        <span className="mx-1 hidden h-4 w-px bg-border sm:block" />
        {props.bound && props.onUnbind ? (
          <Button size="sm" variant="ghost" className="text-accent" onClick={props.onUnbind}>
            <Unlink className="size-3.5" />
            {copy.pages.unbindChat}
          </Button>
        ) : props.onBind ? (
          <Button size="sm" variant="ghost" onClick={props.onBind}>
            <MessageSquare className="size-3.5" />
            {copy.pages.bindChat}
          </Button>
        ) : null}
        <span className="ml-auto text-[11px] tabular-nums text-muted" data-testid="page-save-state">
          {props.state === "saved" ? copy.pages.saved
            : props.state === "saving" ? copy.pages.saving
              : props.state === "dirty" ? copy.pages.dirty
                : props.state === "conflict" ? copy.pages.conflict
                  : copy.pages.error}
          {when ? ` · ${when}` : ""}
          {measure.n ? ` · ${measureLabel}` : ""}
        </span>
        {props.state === "conflict" && props.onReload ? (
          <Button size="sm" variant="ghost" onClick={props.onReload}>{copy.pages.reload}</Button>
        ) : null}
        <Button size="sm" variant="ghost" onClick={props.onSave} disabled={props.state === "saving"}>{copy.pages.save}</Button>
        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <Button size="icon" variant="ghost" className="size-7" aria-label={copy.pages.more}>
              <MoreHorizontal className="size-4" />
            </Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent align="end">
            {props.onNewChild ? (
              <DropdownMenuItem onSelect={() => props.onNewChild?.()}>{copy.pages.newChild}</DropdownMenuItem>
            ) : null}
            {props.onMoveTop && props.parentId ? (
              <DropdownMenuItem onSelect={() => props.onMoveTop?.()}>{copy.pages.moveTop}</DropdownMenuItem>
            ) : null}
            {props.onReload ? (
              <DropdownMenuItem onSelect={() => props.onReload?.()}>{copy.pages.reload}</DropdownMenuItem>
            ) : null}
            {props.onDelete ? (
              <>
                <DropdownMenuSeparator />
                <DropdownMenuItem className="text-danger" onSelect={() => props.onDelete?.()}>
                  <Trash2 className="size-3.5" />
                  {copy.pages.delete}
                </DropdownMenuItem>
              </>
            ) : null}
          </DropdownMenuContent>
        </DropdownMenu>
      </div>
      {props.error ? <div className="px-5 py-1.5 text-[12px] text-danger">{props.error}</div> : null}
      <div className="min-h-0 flex-1 overflow-auto">
        <div className="mx-auto w-full max-w-3xl px-6 pb-16 pt-8 sm:px-10">
          <Input
            className="h-auto border-0 bg-transparent px-0 text-[28px] font-semibold leading-tight tracking-[-0.038em] shadow-none placeholder:text-muted/50"
            value={localTitle}
            aria-label={copy.pages.title}
            placeholder={copy.pages.untitled}
            onChange={(e) => {
              setLocalTitle(e.target.value);
              props.onTitle(e.target.value);
            }}
          />
          <div className="mt-5">
            <EditorContent editor={editor} />
          </div>
        </div>
      </div>
    </div>
  );
}

/** 画一个带提示的格式按钮。props 读取 label、active、children，调用 onClick。 */
function MarkBtn(props: {
  editor: Editor | null;
  label: string;
  active?: boolean;
  onClick: () => void;
  children: ReactNode;
}) {
  return (
    <Tooltip content={props.label} side="bottom">
      <button
        type="button"
        aria-label={props.label}
        aria-pressed={!!props.active}
        className={cn(
          "grid size-7 place-items-center rounded-md",
          props.active ? "bg-lift text-foreground" : "text-muted hover:bg-lift/70 hover:text-foreground",
        )}
        onMouseDown={(e) => {
          e.preventDefault();
          props.onClick();
        }}
      >
        {props.children}
      </button>
    </Tooltip>
  );
}
