import type { CSSProperties } from "react";
import { Toaster as SonnerToaster, type ToasterProps } from "sonner";

const LIGHT_THEMES = new Set(["cinema-light"]);

// Toaster reads the host-injected theme from document.documentElement so
// toasts visually match the rest of the host UI. lib/authToken sets that
// dataset attribute during boot from either the URL ?theme= or the
// X-Continuum-Theme response.
const Toaster = ({ ...props }: ToasterProps) => {
  const theme = typeof document !== "undefined" ? document.documentElement.dataset.theme ?? "" : "";
  const sonnerTheme = LIGHT_THEMES.has(theme) ? "light" : "dark";

  return (
    <SonnerToaster
      theme={sonnerTheme}
      className="toaster group"
      toastOptions={{
        classNames: {
          toast: "!text-[var(--popover-foreground)]",
          title: "!text-[var(--popover-foreground)]",
          description: "!text-[var(--popover-foreground)]",
        },
      }}
      style={
        {
          "--normal-bg": "var(--popover)",
          "--normal-text": "var(--popover-foreground)",
          "--normal-border": "var(--border)",
        } as CSSProperties
      }
      {...props}
    />
  );
};

export { Toaster };
