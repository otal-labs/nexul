import { AlertTriangle } from "lucide-react";

import { errorMessage } from "@/api/client";
import { cn } from "@/lib/utils";

interface ErrorDisplayProps {
  error?: unknown;
  message?: string;
  title?: string;
  className?: string;
}

export const ErrorDisplay = ({ error, message, title = "Something went wrong", className }: ErrorDisplayProps) => (
  <div
    role="alert"
    className={cn("flex flex-col items-center gap-2 rounded-lg border border-border p-8 text-center", className)}
  >
    <span className="flex size-11 items-center justify-center rounded-lg border border-border bg-muted/60">
      <AlertTriangle className="size-5 text-destructive" aria-hidden />
    </span>
    <h2 className="text-lg font-semibold tracking-tight">{title}</h2>
    {message != null && <p className="text-sm text-muted-foreground">{message}</p>}
    {message == null && error != null && <p className="text-sm text-muted-foreground">{errorMessage(error)}</p>}
  </div>
);
