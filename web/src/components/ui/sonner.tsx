import { useShallow } from "zustand/react/shallow";
import { Toaster as Sonner, type ToasterProps } from "sonner";

import { useThemeStore } from "@/stores/themeStore";

// The surface, status icons and motion live in index.css (Toasts), where they can outrank the library's own styles.
const Toaster = ({ ...props }: ToasterProps) => {
  const theme = useThemeStore(useShallow((s) => s.theme));

  return (
    <Sonner
      theme={theme}
      className="toaster group"
      gap={8}
      toastOptions={{
        classNames: {
          description: "group-[.toast]:text-muted-foreground",
          actionButton: "group-[.toast]:bg-brand group-[.toast]:text-brand-foreground group-[.toast]:rounded-md",
          cancelButton: "group-[.toast]:bg-muted group-[.toast]:text-muted-foreground group-[.toast]:rounded-md",
        },
      }}
      {...props}
    />
  );
};

export { Toaster };
