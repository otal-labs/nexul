import { useEffect, useState } from "react";
import { XIcon, type LucideIcon } from "lucide-react";

import type { ActionGate } from "@nexul/client-core/actionState";

import { ActionButton } from "@/components/action/ActionButton";
import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";

interface ConfirmDestroyButtonProps {
  icon: LucideIcon;
  idleLabel: string;
  confirmLabel?: string;
  onConfirm: () => void;
  disabled?: boolean;
  loading?: boolean;
  armMs?: number;
  // Shows the idle label as the button's text instead of the icon, for a row with room for the word; then label names the row.
  labelled?: boolean;
  // The accessible name when it needs more than the visible word, such as which row it acts on.
  label?: string;
  // The permission the delete needs; without it the button is always live.
  gate?: ActionGate;
}

// A worded one is muted until hovered like the other row actions, and pulled out by its padding so its word ends on the row's edge.
const labelledClass = "-mr-3 text-muted-foreground";

// Arms into confirm/cancel instead of firing on first click; auto-reverts so it can't linger as a trap.
export const ConfirmDestroyButton = ({
  icon: Icon,
  idleLabel,
  confirmLabel = "Confirm",
  onConfirm,
  disabled = false,
  loading = false,
  armMs = 4000,
  labelled = false,
  label,
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
      {!armed && gate && labelled && (
        <ActionButton
          {...gate}
          type="button"
          variant="ghost"
          size="sm"
          aria-label={label}
          className={labelledClass}
          liveClassName="hover:text-destructive"
          disabled={disabled}
          loading={loading}
          onClick={() => setArmed(true)}
        >
          {idleLabel}
        </ActionButton>
      )}
      {!armed && gate && !labelled && (
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
          aria-label={label ?? (labelled ? undefined : idleLabel)}
          title={labelled ? undefined : idleLabel}
          className={cn("hover:text-destructive", labelled && labelledClass)}
          disabled={disabled}
          loading={loading}
          onClick={() => setArmed(true)}
        >
          {!labelled && <Icon className="size-4" />}
          {labelled && idleLabel}
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
