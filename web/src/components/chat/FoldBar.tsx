import { ChevronDown, ChevronUp } from "lucide-react";

interface FoldBarProps {
  label: string;
  open: boolean;
  onToggle: () => void;
}

// One hairline row that opens what a long bot post hides, then closes it again.
export const FoldBar = ({ label, open, onToggle }: FoldBarProps) => (
  <button
    type="button"
    onClick={onToggle}
    aria-expanded={open}
    className="flex w-full items-center justify-between gap-2 rounded-md border border-border bg-card px-2.5 py-1.5 text-xs text-muted-foreground transition-colors duration-150 ease-standard hover:text-foreground focus-visible:ring-2 focus-visible:ring-ring focus-visible:outline-none"
  >
    <span>{label}</span>
    {open && <ChevronUp className="size-3.5" aria-hidden />}
    {!open && <ChevronDown className="size-3.5" aria-hidden />}
  </button>
);
