import { ChevronDownIcon } from "lucide-react";

import { sectionLabelClass } from "@/components/SidebarNav";
import { lastInputWasKeyboard, settleIn } from "@/lib/motion";
import { cn } from "@/lib/utils";

interface SidebarFoldHeaderProps {
  label: string;
  open: boolean;
  onToggle: () => void;
}

// A sidebar group's label that folds the group; on opening, the group's [data-fold] rows settle in.
export const SidebarFoldHeader = ({ label, open, onToggle }: SidebarFoldHeaderProps) => (
  <button
    type="button"
    onClick={(event) => {
      const group = event.currentTarget.parentElement;
      onToggle();
      if (!open && !lastInputWasKeyboard()) requestAnimationFrame(() => settleIn(group?.querySelector("[data-fold]")));
    }}
    aria-expanded={open}
    className={cn(
      sectionLabelClass,
      "flex w-full items-center justify-between rounded-md transition-colors duration-150 ease-standard hover:text-foreground",
    )}
  >
    <span>{label}</span>
    <ChevronDownIcon className={cn("size-3.5 transition-transform duration-150 ease-standard", !open && "-rotate-90")} aria-hidden />
  </button>
);
