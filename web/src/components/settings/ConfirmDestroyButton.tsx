import { useEffect, useState } from "react";
import { XIcon, type LucideIcon } from "lucide-react";

import type { ActionGate } from "@nexul/client-core/actionState";

import { ActionButton } from "@/components/action/ActionButton";
import { Button } from "@/components/ui/button";

interface ConfirmDestroyButtonProps {
  icon: LucideIcon;
  idleLabel: string;
  confirmLabel?: string;
  onConfirm: () => void;
  disabled?: boolean;
  loading?: boolean;
  armMs?: number;
  // The permission the delete needs; without it the button is always live.
  gate?: ActionGate;
}

// Arms into confirm/cancel instead of firing on first click; auto-reverts so it can't linger as a trap.
export const ConfirmDestroyButton = ({
  icon: Icon,
  idleLabel,
  confirmLabel = "Confirm",
  onConfirm,
  disabled = false,
  loading = false,
  armMs = 4000,
  gate,
}: ConfirmDestroyButtonProps) => {
  const [armed, setArmed] = useState(false);

  useEffect(() => {
    if (!armed) return;
    const timer = setTimeout(() => setArmed(false), armMs);
    return () => clearTimeout(timer);
  }, [armed, armMs]);

  return (
    <span className="inline-flex items-center gap-1">
      {!armed && gate && (
        <ActionButton
          {...gate}
          type="button"
          variant="ghost"
          size="icon"
          icon={Icon}
          aria-label={idleLabel}
          title={idleLabel}
          className="size-8"
          liveClassName="hover:text-destructive"
          disabled={disabled}
          loading={loading}
          onClick={() => setArmed(true)}
        />
      )}
      {!armed && !gate && (
        <Button
          type="button"
          variant="ghost"
          size="sm"
          aria-label={idleLabel}
          title={idleLabel}
          className="hover:text-destructive"
          disabled={disabled}
          loading={loading}
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
