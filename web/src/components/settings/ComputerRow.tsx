import { RefreshCwIcon, Trash2 } from "lucide-react";

import { PairComputerDialog } from "@/components/pairing/PairComputerDialog";
import { ComputerMCPToken } from "@/components/settings/ComputerMCPToken";
import { ComputerSetupSummary } from "@/components/settings/ComputerSetupSummary";
import { ConfirmDestroyButton } from "@/components/settings/ConfirmDestroyButton";
import { PairComputerForm } from "@/components/settings/PairComputerForm";
import { SettingsStatus, type SettingsStatusTone } from "@/components/settings/SettingsStatus";
import { Button } from "@/components/ui/button";
import { useDeleteComputer } from "@/hooks/PairingHooks";
import { useFormDialog } from "@/hooks/useFormDialog";
import {
  EXPIRY_WARNING_DAYS,
  PairComputerFormSchema,
  harnessLabel,
  stillPairing,
  type Computer,
  type PairComputerFormData,
} from "@/models/Pairing";
import { daysUntil } from "@/utils/TimeUtility";

interface ComputerRowProps {
  computer: Computer;
  // Keeper-held session state ("connected" | "connecting"), absent = none held.
  presence?: string | undefined;
}

// The keeper reports "connecting" while it retries an unreachable computer too, so only connected earns a color.
const connectionOf = (presence: string | undefined): { tone: SettingsStatusTone; text: string } => {
  if (presence === "connected") return { tone: "success", text: "Connected" };
  if (presence === "connecting") return { tone: "muted", text: "Trying to connect" };
  return { tone: "muted", text: "Not connected" };
};

// Expiry warning threshold (EXPIRY_WARNING_DAYS) matches the settings copy's "warning in the final days" wording.
export const ComputerRow = ({ computer, presence }: ComputerRowProps) => {
  const remove = useDeleteComputer();
  const { open: openRepair } = useFormDialog();
  const pairing = stillPairing(computer);
  const days = daysUntil(computer.token_expires_at);
  const expired = !pairing && days <= 0;
  const expiringSoon = !expired && !pairing && days <= EXPIRY_WARNING_DAYS;

  const repair = async () => {
    await openRepair<PairComputerFormData>({
      title: `Re-pair ${computer.name}`,
      description: "Run `t3 pair` on the computer and paste the one-time token it prints.",
      schema: PairComputerFormSchema,
      okLabel: "Re-pair",
      form: <PairComputerForm computer={computer} />,
      formOptions: {
        defaultValues: { name: computer.name, server_url: computer.server_url, token: "" },
      },
    });
  };

  const connection = connectionOf(presence);

  return (
    <li className="space-y-2 bg-card px-3 py-3 transition-colors duration-150 ease-standard hover:bg-accent/40">
      <div className="flex items-start justify-between gap-3">
        <div className="min-w-0 space-y-1">
          <p className="line-clamp-2 text-sm font-medium break-words" title={computer.name}>
            {computer.name}
          </p>
          <p className="flex min-w-0 flex-wrap items-center gap-x-3 gap-y-0.5">
            {!pairing && <SettingsStatus tone={connection.tone}>{connection.text}</SettingsStatus>}
            {pairing && <SettingsStatus tone="warning">Pairing in progress</SettingsStatus>}
            {expired && <SettingsStatus tone="destructive">Pairing expired, acts as unpaired</SettingsStatus>}
            {expiringSoon && <SettingsStatus tone="warning">Pairing expires in {days}d</SettingsStatus>}
          </p>
          <p className="truncate font-mono text-xs text-muted-foreground tabular-nums" title={computer.server_url}>
            {computer.server_url} · {harnessLabel(computer.kind)} {computer.harness_version}
          </p>
        </div>
        <span className="flex shrink-0 items-center gap-1">
          {/* Mounted past pairing so the refetch that flips the row can't close the wizard as it reaches Set up. */}
          <PairComputerDialog
            existing={computer}
            trigger={
              pairing && (
                <Button type="button" variant="ghost" size="sm">
                  <RefreshCwIcon className="size-4" />
                  Pair
                </Button>
              )
            }
          />
          {!pairing && (
            <Button type="button" variant="ghost" size="sm" onClick={repair}>
              <RefreshCwIcon className="size-4" />
              Re-pair
            </Button>
          )}
          <ConfirmDestroyButton
            icon={Trash2}
            idleLabel="Remove"
            loading={remove.isPending}
            onConfirm={() => remove.mutate(computer)}
          />
        </span>
      </div>
      {!pairing && <ComputerSetupSummary computer={computer} />}
      <ComputerMCPToken computerId={computer.id} />
    </li>
  );
};
