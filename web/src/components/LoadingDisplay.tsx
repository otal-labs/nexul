import { Loader2 } from "lucide-react";

import { cn } from "@/lib/utils";

interface LoadingDisplayProps {
  label?: string;
  className?: string;
}

export const LoadingDisplay = ({ label = "Loading", className }: LoadingDisplayProps) => (
  <div
    role="status"
    className={cn("animate-in fade-in-0 flex items-center justify-center gap-2 p-8 text-muted-foreground duration-150 ease-out", className)}
  >
    <Loader2 className="size-4 animate-spin motion-reduce:animate-none" aria-hidden />
    <span className="text-sm">{label}</span>
  </div>
);
