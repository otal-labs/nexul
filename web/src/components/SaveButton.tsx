import { useEffect, useState } from "react";
import { CheckIcon } from "lucide-react";

import { Button, type ButtonProps } from "@/components/ui/button";
import { cn } from "@/lib/utils";

const SAVED_MS = 1600;

interface SaveButtonProps extends ButtonProps {
  // The mutation's submittedAt once it succeeded; a new value replays the confirmation.
  savedAt: number | undefined;
}

// A Save that answers where it was pressed: for a moment after a save lands it reads "Saved" with a check.
export const SaveButton = ({ savedAt, children, className, ...props }: SaveButtonProps) => {
  const [expired, setExpired] = useState<number | undefined>(undefined);
  useEffect(() => {
    if (!savedAt) return;
    const timer = window.setTimeout(() => setExpired(savedAt), SAVED_MS);
    return () => window.clearTimeout(timer);
  }, [savedAt]);
  const saved = !!savedAt && expired !== savedAt;

  return (
    <Button className={cn("save-button data-saved:disabled:opacity-100", className)} data-saved={saved || undefined} {...props}>
      <span className="grid place-items-center">
        <span className="save-button__label [grid-area:1/1]">{children}</span>
        <span aria-hidden className="save-button__done inline-flex items-center gap-1.5 [grid-area:1/1]">
          <CheckIcon className="size-4" />
          Saved
        </span>
      </span>
      <span className="sr-only" role="status">
        {saved ? "Saved" : ""}
      </span>
    </Button>
  );
};
