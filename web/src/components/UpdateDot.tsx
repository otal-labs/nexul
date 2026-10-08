import { cn } from "@/lib/utils";

interface UpdateDotProps {
  // Read after the owner's own name by screen readers, e.g. "skills out of date"; omit when the owner's aria-label already says it.
  label?: string | undefined;
  className?: string;
}

// The yellow dot that says something behind this control wants updating; its owner must be `relative`.
export const UpdateDot = ({ label, className }: UpdateDotProps) => (
  <>
    <span className={cn("absolute top-1 right-1 size-1.5 rounded-full bg-warning ring-2 ring-surface-2", className)} aria-hidden />
    {label && <span className="sr-only">, {label}</span>}
  </>
);
