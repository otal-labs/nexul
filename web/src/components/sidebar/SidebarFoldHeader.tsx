import { ChevronDownIcon } from "lucide-react";
import type { ReactNode } from "react";

import { sectionLabelClass } from "@/components/SidebarNav";
import { lastInputWasKeyboard, settleIn } from "@/lib/motion";
import { cn } from "@/lib/utils";

interface SidebarFoldHeaderProps {
  label: string;
  open: boolean;
  onToggle: () => void;
  // A create action trailing the chevron, outside the fold button.
  action?: ReactNode;
}

// A sidebar group's label that folds the group; on opening, the group's [data-fold] rows settle in.
export const SidebarFoldHeader = ({ label, open, onToggle, action }: SidebarFoldHeaderProps) => (
  <div className={cn(sectionLabelClass, "flex items-center gap-1 px-0", action && "pr-1")}>
    <button
      type="button"
      onClick={(event) => {
        const group = event.currentTarget.parentElement?.parentElement;
        onToggle();
        if (!open && !lastInputWasKeyboard()) requestAnimationFrame(() => settleIn(group?.querySelector("[data-fold]")));
      }}
      aria-expanded={open}
      className="flex min-w-0 flex-1 items-center justify-between rounded-md px-2.5 transition-colors duration-150 ease-standard hover:text-foreground"
    >
      <span>{label}</span>
      <ChevronDownIcon className={cn("size-3.5 transition-transform duration-150 ease-standard", !open && "-rotate-90")} aria-hidden />
    </button>
    {action}
  </div>
);
