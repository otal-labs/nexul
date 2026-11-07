import { useQueryClient } from "@tanstack/react-query";
import { isAxiosError } from "axios";
import { AlertTriangle } from "lucide-react";

import { retryUnlessClientError } from "@nexul/client-core/queryRetry";

import { errorMessage } from "@/api/client";
import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";

interface ErrorDisplayProps {
  error?: unknown;
  message?: string;
  title?: string;
  className?: string;
}

// Only a failure a second attempt could fix offers one: no answer from the server, or a 5xx; a 4xx is final.
const worthRetrying = (error: unknown) => isAxiosError(error) && retryUnlessClientError(0, error.response?.status);

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
    {worthRetrying(error) && <RetryButton />}
  </div>
);

// Refetches every query on screen that failed, which is what the panel above it is reporting.
const RetryButton = () => {
  const queryClient = useQueryClient();
  return (
    <Button
      variant="outline"
      size="sm"
      className="mt-2"
      onClick={() => void queryClient.refetchQueries({ type: "active", predicate: (query) => query.state.status === "error" })}
    >
      Try again
    </Button>
  );
};
