import React from "react";
import ReactDOM from "react-dom/client";
import { Toaster } from "sonner";
import App from "./App";
import { AppErrorBoundary } from "./features/AppErrorBoundary";
import { installFrontendLogBridge } from "./lib/diag-bridge";
import { ThemeProvider, useTheme } from "./lib/theme";
import { TooltipProvider } from "./components/ui/tooltip";
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
          toast: "border-border bg-popover text-foreground shadow-[var(--shadow-popover)]",
        },
      }}
    />
  );
}

ReactDOM.createRoot(document.getElementById("root") as HTMLElement).render(
  <React.StrictMode>
    <ThemeProvider>
      <TooltipProvider>
        <AppErrorBoundary>
          <App />
        </AppErrorBoundary>
        <ThemedToaster />
      </TooltipProvider>
    </ThemeProvider>
  </React.StrictMode>,
);
