import { useEffect, useRef, useState } from "react";
import Placeholder from "@tiptap/extension-placeholder";
import { EditorContent, useEditor } from "@tiptap/react";
import StarterKit from "@tiptap/starter-kit";
import { useCopy } from "../../lib/i18n";
import { Button } from "../../components/ui/button";
import { Input } from "../../components/ui/input";
import { htmlToMd, mdToHtml } from "./markdown";

export type SaveState = "saved" | "saving" | "dirty" | "conflict" | "error";

export function PageEditor(props: {
  title: string;
  content: string;
  revision: number;
  state: SaveState;
  error?: string;
  onTitle: (t: string) => void;
  onContent: (md: string) => void;
  onSave: () => void;
  onReload?: () => void;
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
    editorProps: { attributes: { class: "page-editor min-h-[50vh] max-w-3xl px-1 py-2 text-[15px] leading-7 focus:outline-none" } },
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
  }, [editor, props.revision]);

  useEffect(() => {
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
    const leave = (e: BeforeUnloadEvent) => {
      if (!dirty) return;
      e.preventDefault();
      e.returnValue = "";
    };
    window.addEventListener("beforeunload", leave);
    return () => window.removeEventListener("beforeunload", leave);
  }, [props.state]);

  return (
    <div className="flex min-h-0 flex-1 flex-col" data-testid="page-editor">
      <div className="flex items-center gap-2 border-b border-border/50 px-4 py-2">
        <Input
          className="h-9 flex-1 border-0 bg-transparent text-[18px] font-medium shadow-none"
          value={localTitle}
          aria-label={copy.pages.title}
          onChange={(e) => {
            setLocalTitle(e.target.value);
            props.onTitle(e.target.value);
          }}
        />
        <span className="text-[11px] text-muted" data-testid="page-save-state">
          {props.state === "saved" ? copy.pages.saved
            : props.state === "saving" ? copy.pages.saving
              : props.state === "dirty" ? copy.pages.dirty
                : props.state === "conflict" ? copy.pages.conflict
                  : copy.pages.error}
        </span>
        {props.state === "conflict" && props.onReload ? (
          <Button size="sm" variant="ghost" onClick={props.onReload}>{copy.pages.reload}</Button>
        ) : null}
        <Button size="sm" onClick={props.onSave} disabled={props.state === "saving"}>{copy.pages.save}</Button>
      </div>
      {props.error ? <div className="px-4 py-1 text-[12px] text-danger">{props.error}</div> : null}
      <div className="min-h-0 flex-1 overflow-auto px-4 py-3">
        <EditorContent editor={editor} />
      </div>
    </div>
  );
}
