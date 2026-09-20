import { AlertTriangle } from "lucide-react";

import { errorMessage } from "@/api/client";
import { cn } from "@/lib/utils";

interface ErrorDisplayProps {
  error?: unknown;
  title?: string;
  className?: string;
}

export const ErrorDisplay = ({ error, title = "Something went wrong", className }: ErrorDisplayProps) => (
  <div
    role="alert"
    className={cn("flex flex-col items-center gap-2 rounded-lg border border-destructive/30 bg-destructive/5 p-8 text-center", className)}
  >
    <span className="flex size-11 items-center justify-center rounded-lg border border-destructive/30 bg-destructive/10">
      <AlertTriangle className="size-5 text-destructive" aria-hidden />
    </span>
    <h2 className="text-lg font-semibold tracking-tight">{title}</h2>
    {error != null && <p className="text-sm text-muted-foreground">{errorMessage(error)}</p>}
  </div>
);
