import { useShallow } from "zustand/react/shallow";
import { Toaster as Sonner, type ToasterProps } from "sonner";

import { useThemeStore } from "@/stores/themeStore";

const Toaster = ({ ...props }: ToasterProps) => {
  const theme = useThemeStore(useShallow((s) => s.theme));

  return (
    <Sonner
      theme={theme}
      className="toaster group"
      toastOptions={{
        classNames: {
          toast:
            "group toast group-[.toaster]:rounded-lg group-[.toaster]:border group-[.toaster]:bg-card group-[.toaster]:text-foreground group-[.toaster]:shadow-elevated",
          description: "group-[.toast]:text-muted-foreground",
          actionButton:
            "group-[.toast]:bg-primary group-[.toast]:text-primary-foreground group-[.toast]:rounded-md",
          cancelButton:
            "group-[.toast]:bg-muted group-[.toast]:text-muted-foreground group-[.toast]:rounded-md",
          success: "group-[.toast]:text-success",
          error: "group-[.toast]:text-destructive",
          warning: "group-[.toast]:text-warning",
          info: "group-[.toast]:text-info",
        },
      }}
      {...props}
    />
  );
};

export { Toaster };
