// 本文件拥有用 srcdoc 打开的预览框和它的 sandbox。不拥有 HTML 怎么包成文档，那一步在 asPreviewDocument。Transcript 和 FilePreview 传入 html。

import { asPreviewDocument } from "../../lib/html-preview";
import { cn } from "../../lib/utils";

/**
 * SandboxedFrame 在 allowScripts 为真时允许脚本、表单、模态和指针锁定，否则只允许表单。fill 为真时撑满父级，否则高度 28rem。文档交给 asPreviewDocument，并把 scripts 传进去。
 * html 传给 asPreviewDocument。
 * title 写到 iframe 的 title。
 */
export function SandboxedFrame({
  html,
  title,
  fill,
  allowScripts,
}: {
  html: string;
  title: string;
  fill?: boolean;
  allowScripts?: boolean;
}) {
  const scripts = !!allowScripts;
  return (
    <iframe
      title={title}
      sandbox={scripts ? "allow-scripts allow-forms allow-modals allow-pointer-lock" : "allow-forms"}
      className={cn(
        "block w-full min-w-0 max-w-full bg-background",
        fill ? "h-full min-h-0 w-full flex-1 rounded-none border-0" : "mt-2 h-[28rem] rounded-lg border border-border/70",
      )}
      style={{ colorScheme: "light", pointerEvents: "auto" }}
      srcDoc={asPreviewDocument(html, { scripts })}
      data-testid="html-preview"
    />
  );
}
