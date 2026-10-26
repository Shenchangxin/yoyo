// 本文件拥有应用外框、主栏和侧栏卡片的骨架。不拥有里面的会话或页面。App 使用 AppFrame 和 MainColumn，ThreadRail 使用 SidebarCard。

import type { ComponentProps, ReactNode } from "react";
import { cn } from "../../lib/utils";

/** AppFrame 铺壁纸和颗粒，overlay 叠在内容之前。内容区标成 data-app-frame="content"，并带窗口拖拽。children 作为子节点渲染。className 传给 cn。props 展开到 div。 */
export function AppFrame({
  children,
  overlay,
  className,
  ...props
}: ComponentProps<"div"> & { overlay?: ReactNode }) {
  return (
    <div
      className={cn("relative isolate flex h-full w-full flex-col overflow-hidden bg-background text-foreground", className)}
      {...props}
    >
      <div className="app-wallpaper" aria-hidden />
      <div className="app-grain" aria-hidden />
      {overlay}
      <div className="chrome drag relative z-10 flex h-full min-h-0 flex-1 items-stretch overflow-hidden" data-app-frame="content">
        {children}
      </div>
    </div>
  );
}

/** MainColumn 是主舞台，id 为 main-stage，tabIndex 为 -1，本身不参与窗口拖拽。children 作为子节点渲染。className 传给 cn。 */
export function MainColumn({ children, className }: { children: ReactNode; className?: string }) {
  return (
    <div
      className={cn(
        "no-drag relative flex h-full min-h-0 min-w-0 flex-1 flex-col overflow-hidden overscroll-none bg-transparent",
        className,
      )}
      id="main-stage"
      tabIndex={-1}
    >
      {children}
    </div>
  );
}

/** SidebarCard 是一条玻璃侧栏，高度撑满，本身不参与窗口拖拽。children 作为子节点渲染。className 传给 cn。 */
export function SidebarCard({ children, className }: { children: ReactNode; className?: string }) {
  return (
    <div
      className={cn(
        "glass-chrome no-drag flex h-full min-h-0 flex-col overflow-hidden",
        className,
      )}
    >
      {children}
    </div>
  );
}
