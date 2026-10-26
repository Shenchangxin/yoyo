// 本文件拥有面板之间的拖拽分隔。不拥有面板里的内容。App 把它放在侧栏和主栏之间。

import { Separator } from "react-resizable-panels";

// ResizeHandle 是一条 1.5 单位宽的发光分隔。
export function ResizeHandle() {
  return <Separator className="resize-glow relative z-10 w-1.5" />;
}
