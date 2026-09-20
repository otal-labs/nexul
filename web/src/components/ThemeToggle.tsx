import { Moon, Sun } from "lucide-react";
import { useShallow } from "zustand/react/shallow";

import { Button } from "@/components/ui/button";
import { useThemeStore } from "@/stores/themeStore";

export const ThemeToggle = () => {
  const { theme, toggleTheme } = useThemeStore(
    useShallow((s) => ({ theme: s.theme, toggleTheme: s.toggleTheme })),
  );

  return (
    <Button
      variant="ghost"
      size="icon"
      onClick={toggleTheme}
      aria-label={theme === "dark" ? "Switch to light theme" : "Switch to dark theme"}
    >
      {theme === "dark" && <Sun className="size-4" />}
      {theme !== "dark" && <Moon className="size-4" />}
    </Button>
  );
};
