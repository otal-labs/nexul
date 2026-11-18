import { ChevronDown } from "lucide-react";

import { cn } from "@/lib/utils";

interface RoleFoldButtonProps {
  open: boolean;
  onToggle: () => void;
  label: string;
  className?: string;
}

// The ghost line that opens a section's folded areas in place, instantly.
export const RoleFoldButton = ({ open, onToggle, label, className }: RoleFoldButtonProps) => (
  <button
    type="button"
    aria-expanded={open}
    onClick={onToggle}
    className={cn("inline-flex h-8 items-center gap-1 rounded-md px-2 text-xs text-muted-foreground transition-colors duration-150 ease-standard hover:bg-accent hover:text-foreground", className)}
  >
    <ChevronDown aria-hidden className={cn("size-3.5", open && "rotate-180")} />
    {label}
  </button>
);
