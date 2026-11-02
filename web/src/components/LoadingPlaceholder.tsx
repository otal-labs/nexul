import type { HTMLAttributes } from "react";

import { LOADER_DELAY_MS } from "@/lib/motion";
import { cn } from "@/lib/utils";

interface LoadingPlaceholderProps extends HTMLAttributes<HTMLDivElement> {
  label: string;
}

// A page whose layout is known loads in its own shape, held back like LoadingDisplay so a fast load shows nothing.
export const LoadingPlaceholder = ({ label, className, children, ...props }: LoadingPlaceholderProps) => (
  <div
    {...props}
    role="status"
    className={cn("animate-in fade-in-0 duration-150 ease-out", className)}
    style={{ animationDelay: `${LOADER_DELAY_MS}ms`, animationFillMode: "both" }}
  >
    <span className="sr-only">{label}</span>
    {children}
  </div>
);

interface PlaceholderBarProps {
  className?: string;
}

export const PlaceholderBar = ({ className }: PlaceholderBarProps) => (
  <span aria-hidden className={cn("block rounded-md bg-muted/40 animate-pulse motion-reduce:animate-none", className)} />
);
