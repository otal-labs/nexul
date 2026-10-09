import { ChevronDown } from "lucide-react";

import { cn } from "@/lib/utils";

interface FoldBarProps {
  label: string;
  open: boolean;
  onToggle: () => void;
  className?: string;
}

// Opens what a long bot post hides, then closes it again.
export const FoldBar = ({ label, open, onToggle, className }: FoldBarProps) => (
  <button
    type="button"
    onClick={onToggle}
    aria-expanded={open}
    className={cn(
      "-mx-1.5 inline-flex items-center gap-1 rounded-md px-1.5 py-0.5 text-xs font-medium text-muted-foreground transition-colors duration-150 ease-standard hover:bg-accent hover:text-foreground focus-visible:ring-2 focus-visible:ring-ring focus-visible:outline-none",
      className,
    )}
  >
    {label}
    <ChevronDown aria-hidden className={cn("size-3.5 transition-transform duration-150 ease-standard", open && "rotate-180")} />
  </button>
);
