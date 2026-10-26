// 本文件拥有根节点挂载、主题和提示条。不拥有 App 里的界面。index.html 加载它。伴侣面不挂 TooltipProvider 和 Toaster。

import React from "react";
import ReactDOM from "react-dom/client";
import { Toaster } from "sonner";
import App from "./App";
import { AppErrorBoundary } from "./features/AppErrorBoundary";
import { installFrontendLogBridge } from "./lib/diag-bridge";
import { ThemeProvider, useTheme } from "./lib/theme";
import { TooltipProvider } from "./components/ui/tooltip";
import { isCompanionSurface } from "./lib/popout";
import { applyOsClass } from "./lib/chrome";
import { installInputGuards } from "./lib/input-guard";
import { useMedia } from "./lib/media";
import "./styles.css";

installFrontendLogBridge();
applyOsClass();
installInputGuards();

// 按当前主题和窄屏选择提示条位置。窄于 799px 时改到顶部居中，offset 为 72，否则右下角、offset 为 24。
function ThemedToaster() {
  const { resolved } = useTheme();
  const narrow = useMedia("(max-width: 799px)");
  return (
    <Toaster
      theme={resolved}
      position={narrow ? "top-center" : "bottom-right"}
      offset={narrow ? 72 : 24}
      closeButton
      toastOptions={{
        classNames: {
          toast: "yoyo-toast rounded-xl border-border bg-popover text-foreground shadow-[var(--shadow-popover)]",
        },
      }}
    />
  );
}

const companion = isCompanionSurface();

ReactDOM.createRoot(document.getElementById("root") as HTMLElement).render(
  <React.StrictMode>
    <ThemeProvider>
      {companion ? (
        <AppErrorBoundary>
          <App />
        </AppErrorBoundary>
      ) : (
        <TooltipProvider>
          <AppErrorBoundary>
            <App />
          </AppErrorBoundary>
          <ThemedToaster />
        </TooltipProvider>
      )}
    </ThemeProvider>
  </React.StrictMode>,
);
