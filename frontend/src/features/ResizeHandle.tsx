// 本文件拥有面板之间的拖拽分隔。不拥有面板里的内容。App 把它放在侧栏和主栏之间。

import { Separator } from "react-resizable-panels";

// ResizeHandle 平时是一条发丝分界，热区比线宽，拖动时中间收成一截短把手。
export function ResizeHandle() {
  return <Separator className="resize-glow" />;
}
