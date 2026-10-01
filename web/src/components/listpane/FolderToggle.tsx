import type { ReactNode } from "react";
import { ChevronRightIcon, FolderIcon, FolderOpenIcon } from "lucide-react";

import { cn } from "@/lib/utils";

interface FolderToggleProps {
  name: string;
  open: boolean;
  onToggle: () => void;
  /** Trailing muted meta, a count or a summary of what the folder holds. */
  meta: ReactNode;
  metaClassName?: string;
  className?: string;
}

// A folder row's label: chevron, folder icon, name in the group-label style, and its meta trailing right.
export const FolderToggle = ({ name, open, onToggle, meta, metaClassName, className }: FolderToggleProps) => (
  <button
    type="button"
    onClick={onToggle}
    aria-expanded={open}
    className={cn(
      "flex min-w-0 flex-1 items-center gap-1.5 self-stretch font-mono text-[11px] font-medium tracking-[0.08em] text-muted-foreground uppercase outline-none hover:text-foreground focus-visible:text-foreground",
      className,
    )}
  >
    <ChevronRightIcon className={cn("size-3.5 shrink-0 transition-transform duration-150 ease-standard", open && "rotate-90")} aria-hidden />
    {open && <FolderOpenIcon className="size-3.5 shrink-0" aria-hidden />}
    {!open && <FolderIcon className="size-3.5 shrink-0" aria-hidden />}
    <span className="truncate">{name}</span>
    <span className={cn("ml-auto shrink-0 tabular-nums", metaClassName)}>{meta}</span>
  </button>
);
