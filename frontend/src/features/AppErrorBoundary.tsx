import { Component, type ErrorInfo, type ReactNode } from "react";
import { catalog } from "../lib/copy";
import { reportRenderer } from "../lib/diag-bridge";

type State = { err: Error | null };

export class AppErrorBoundary extends Component<{ children: ReactNode }, State> {
  state: State = { err: null };

  static getDerivedStateFromError(err: Error): State {
    return { err };
  }

  componentDidCatch(err: Error, info: ErrorInfo) {
    console.error(err, info.componentStack);
    reportRenderer({
      level: "error",
      msg: err.message,
      stack: `${err.stack || ""}\n${info.componentStack || ""}`,
      source: "ErrorBoundary",
    });
  }

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
