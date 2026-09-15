import React from "react";
import ReactDOM from "react-dom/client";
import { Toaster } from "sonner";
import App from "./App";
import { ThemeProvider, useTheme } from "./lib/theme";
import "./styles.css";

function ThemedToaster() {
  const { resolved } = useTheme();
  return (
    <Toaster
      theme={resolved}
      position="bottom-right"
      closeButton
      toastOptions={{
        classNames: {
          toast: "border-border bg-panel text-foreground shadow-lg",
        },
      }}
    />
  );
}

ReactDOM.createRoot(document.getElementById("root") as HTMLElement).render(
  <React.StrictMode>
    <ThemeProvider>
      <App />
      <ThemedToaster />
    </ThemeProvider>
  </React.StrictMode>,
);
