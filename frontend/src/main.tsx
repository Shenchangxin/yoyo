import React from "react";
import ReactDOM from "react-dom/client";
import { Toaster } from "sonner";
import App from "./App";
import { AppErrorBoundary } from "./features/AppErrorBoundary";
import { installFrontendLogBridge } from "./lib/diag-bridge";
import { ThemeProvider, useTheme } from "./lib/theme";
import { TooltipProvider } from "./components/ui/tooltip";
import { isCompanionSurface } from "./lib/popout";
import "./styles.css";

installFrontendLogBridge();

function ThemedToaster() {
  const { resolved } = useTheme();
  return (
    <Toaster
      theme={resolved}
      position="bottom-right"
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
