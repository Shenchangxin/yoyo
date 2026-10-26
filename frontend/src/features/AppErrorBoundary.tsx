// 本文件拥有渲染崩溃后的提示和重试。不拥有业务界面。main 把它包在 App 外面。

import { Component, type ErrorInfo, type ReactNode } from "react";
import { catalog } from "../lib/copy";
import { reportRenderer } from "../lib/diag-bridge";

// err 为 null 时照常渲染子节点。
type State = { err: Error | null };

// AppErrorBoundary 抓住子树错误后换成提示，重试会整页刷新。
export class AppErrorBoundary extends Component<{ children: ReactNode }, State> {
  state: State = { err: null };

  /** getDerivedStateFromError 只记下错误，下一轮渲染改走提示。err 写入字段 err。 */
  static getDerivedStateFromError(err: Error): State {
    return { err };
  }

  /** componentDidCatch 把错误和组件栈打到控制台，并交给 reportRenderer，source 为 ErrorBoundary。err 读取 message、stack，传给 console.error。info 读取 componentStack。 */
  componentDidCatch(err: Error, info: ErrorInfo) {
    console.error(err, info.componentStack);
    reportRenderer({
      level: "error",
      msg: err.message,
      stack: `${err.stack || ""}\n${info.componentStack || ""}`,
      source: "ErrorBoundary",
    });
  }

  // render 没有错误时返回子节点。有错误时显示文案和这条错误的 message。
  render() {
    if (!this.state.err) return this.props.children;
    const copy = catalog();
    return (
      <div className="flex h-full items-center justify-center bg-background p-8 text-foreground">
        <div className="max-w-md text-center">
          <p className="text-[15px] font-medium">{copy.app.errorNotice}</p>
          <p className="mt-2 text-[13px] text-muted">{this.state.err.message}</p>
          <button type="button" className="mt-4 text-[13px] underline" onClick={() => location.reload()}>
            {copy.app.retry}
          </button>
        </div>
      </div>
    );
  }
}
