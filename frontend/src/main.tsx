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
