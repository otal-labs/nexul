import { useRef, type ReactNode } from "react";

import { useSwapEntrance } from "@/hooks/useSwapEntrance";
import { cn } from "@/lib/utils";

interface SettingsShellProps {
  nav: ReactNode;
  // The open section; its content swaps in from the side of the nav item it was picked from.
  section: string;
  children: ReactNode;
  className?: string;
}

// The settings-style page body (Your settings, Configuration, Project settings, Stack): the section nav beside one section's content.
export const SettingsShell = ({ nav, section, children, className }: SettingsShellProps) => {
  const root = useRef<HTMLDivElement>(null);
  const content = useRef<HTMLDivElement>(null);
  useSwapEntrance(
    section,
    () => [...(root.current?.querySelectorAll("nav a") ?? [])].findIndex((link) => link.getAttribute("aria-current") === "page"),
    () => content.current,
    // The nav is a column from 1024px and a row below it, so the content follows the axis the pointer travelled.
    () => (window.matchMedia?.("(min-width: 1024px)").matches ? "y" : "x"),
  );
  return (
    <div ref={root} className={cn("flex flex-col gap-6 lg:flex-row lg:items-start lg:gap-8", className)}>
      {nav}
      <div ref={content} className="min-w-0 flex-1 space-y-6">
        {children}
      </div>
    </div>
  );
};
