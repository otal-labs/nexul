import { RefreshCwIcon, Trash2 } from "lucide-react";

import { ConfirmDestroyButton } from "@/components/settings/ConfirmDestroyButton";
import { PairComputerForm } from "@/components/settings/PairComputerForm";
import { Button } from "@/components/ui/button";
import { useDeleteComputer } from "@/hooks/PairingHooks";
import { useFormDialog } from "@/hooks/useFormDialog";
import { EXPIRY_WARNING_DAYS, PairComputerFormSchema, harnessLabel, type Computer, type PairComputerFormData } from "@/models/Pairing";
import { daysUntil } from "@/utils/TimeUtility";

interface ComputerRowProps {
  computer: Computer;
  // Keeper-held session state ("connected" | "connecting"), absent = none held.
  presence?: string | undefined;
}

// Expiry warning threshold (EXPIRY_WARNING_DAYS) matches the settings copy's "warning in the final days" wording.
export const ComputerRow = ({ computer, presence }: ComputerRowProps) => {
  const remove = useDeleteComputer();
  const { open: openRepair } = useFormDialog();
  const days = daysUntil(computer.token_expires_at);
  const expired = days <= 0;
  const expiringSoon = !expired && days <= EXPIRY_WARNING_DAYS;

  const repair = async () => {
    await openRepair<PairComputerFormData>({
      title: `Re-pair ${computer.name}`,
      description: "Run `t3 pair` on the machine again, then paste the fresh one-time token.",
      schema: PairComputerFormSchema,
      okLabel: "Re-pair",
      form: <PairComputerForm computerId={computer.id} />,
      formOptions: {
        defaultValues: { name: computer.name, server_url: computer.server_url, token: "" },
      },
    });
  };

  const dot =
    presence === "connected"
      ? { className: "bg-success", label: "Connected" }
      : presence === "connecting"
        ? { className: "bg-warning animate-pulse", label: "Connecting\u2026" }
        : { className: "bg-muted-foreground/40", label: "Not connected" };

  return (
    <li className="flex items-center justify-between gap-3 bg-card px-3 py-3 transition-colors duration-150 ease-standard hover:bg-accent/40">
      <div className="min-w-0">
        <p className="flex items-center gap-1.5 truncate text-sm font-medium">
          <span title={dot.label} aria-label={dot.label} className={`inline-block size-2 shrink-0 rounded-full ${dot.className}`} />
          {computer.name}
          {expired && (
            <span className="ml-2 rounded bg-destructive/15 px-1.5 py-0.5 text-xs text-destructive">
              expired — acts as unpaired
            </span>
          )}
          {expiringSoon && (
            <span className="ml-2 rounded bg-warning/15 px-1.5 py-0.5 text-xs text-warning">
              expires in {days}d
            </span>
          )}
        </p>
        <p className="truncate font-mono text-xs text-muted-foreground tabular-nums">
          {computer.server_url} · {harnessLabel(computer.kind)} {computer.harness_version}
        </p>
      </div>
      <span className="flex shrink-0 items-center gap-1">
        <Button type="button" variant="ghost" size="sm" onClick={repair}>
          <RefreshCwIcon className="size-4" />
          Re-pair
        </Button>
        <ConfirmDestroyButton
          icon={Trash2}
          idleLabel="Remove"
          disabled={remove.isPending}
          onConfirm={() => remove.mutate(computer.id)}
        />
      </span>
    </li>
  );
};
