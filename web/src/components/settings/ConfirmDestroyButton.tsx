import { useEffect, useState, type ComponentType } from "react";
import { XIcon } from "lucide-react";

import { Button } from "@/components/ui/button";

interface ConfirmDestroyButtonProps {
  icon: ComponentType<{ className?: string }>;
  idleLabel: string;
  confirmLabel?: string;
  onConfirm: () => void;
  disabled?: boolean;
  armMs?: number;
}

// Arms into confirm/cancel instead of firing on first click; auto-reverts so it can't linger as a trap.
export const ConfirmDestroyButton = ({
  icon: Icon,
  idleLabel,
  confirmLabel = "Confirm",
  onConfirm,
  disabled = false,
  armMs = 4000,
}: ConfirmDestroyButtonProps) => {
  const [armed, setArmed] = useState(false);

  useEffect(() => {
    if (!armed) return;
    const timer = setTimeout(() => setArmed(false), armMs);
    return () => clearTimeout(timer);
  }, [armed, armMs]);

  return (
    <span className="inline-flex items-center gap-1">
      {!armed && (
        <Button
          type="button"
          variant="ghost"
          size="sm"
          aria-label={idleLabel}
          title={idleLabel}
          className="hover:text-destructive"
          disabled={disabled}
          onClick={() => setArmed(true)}
        >
          <Icon className="size-4" />
        </Button>
      )}
      {armed && (
        <span className="animate-in fade-in-0 zoom-in-95 inline-flex items-center gap-1 duration-150 ease-out">
          <Button
            type="button"
            variant="destructive"
            size="sm"
            onClick={() => {
              setArmed(false);
              onConfirm();
            }}
          >
            {confirmLabel}
          </Button>
          <Button
            type="button"
            variant="ghost"
            size="sm"
            aria-label="Cancel"
            title="Cancel"
            onClick={() => setArmed(false)}
          >
            <XIcon className="size-4" />
          </Button>
        </span>
      )}
    </span>
  );
};
