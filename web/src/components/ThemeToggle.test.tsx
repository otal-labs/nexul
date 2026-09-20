import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it } from "vitest";

import { ThemeToggle } from "@/components/ThemeToggle";
import { ThemeName } from "@/enums/Theme";
import { useThemeStore } from "@/stores/themeStore";

describe("ThemeToggle", () => {
  beforeEach(() => {
    document.documentElement.classList.remove("dark");
    useThemeStore.setState({ theme: ThemeName.Light });
    localStorage.clear();
  });

  it("toggles to dark when clicked", async () => {
    const user = userEvent.setup();
    render(<ThemeToggle />);
    await user.click(screen.getByRole("button", { name: "Switch to dark theme" }));
    expect(useThemeStore.getState().theme).toBe(ThemeName.Dark);
  });

  it("updates the aria-label to reflect the current theme", () => {
    useThemeStore.setState({ theme: ThemeName.Dark });
    render(<ThemeToggle />);
    expect(screen.getByRole("button", { name: "Switch to light theme" })).toBeInTheDocument();
  });
});
