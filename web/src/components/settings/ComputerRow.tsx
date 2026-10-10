import { useState } from "react";
import { ChevronRight, RefreshCw, Wrench } from "lucide-react";
import { useSearchParams } from "react-router";

import { PairComputerDialog } from "@/components/pairing/PairComputerDialog";
import { ComputerDetails } from "@/components/settings/ComputerDetails";
import { PairComputerForm } from "@/components/settings/PairComputerForm";
import { RowActionsMenu, type RowAction } from "@/components/settings/RowActionsMenu";
import { SettingsStatus, type SettingsStatusTone } from "@/components/settings/SettingsStatus";
import { Button } from "@/components/ui/button";
import { useFetchComputerSetup, useUpdateSkills } from "@/hooks/ComputerSetupHooks";
import { useDeleteComputer } from "@/hooks/PairingHooks";
import { useConfirmationDialog } from "@/hooks/useConfirmationDialog";
import { useFormDialog } from "@/hooks/useFormDialog";
import {
  EXPIRY_WARNING_DAYS,
  PairComputerFormSchema,
  onlySkillsOutdated,
  setupSummary,
  stillPairing,
  type Computer,
  type PairComputerFormData,
} from "@/models/Pairing";
import { cn } from "@/lib/utils";
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

type Primary = "pair" | "repair" | "update" | "setup" | undefined;

// One computer: its name and state on two lines, at most one action for what needs doing, the rest in its menu and under its fold.
export const ComputerRow = ({ computer, presence }: ComputerRowProps) => {
  const [searchParams, setSearchParams] = useSearchParams();
  const pairing = stillPairing(computer);
  const { data: setup } = useFetchComputerSetup(pairing ? "" : computer.id);
  const remove = useDeleteComputer();
  const update = useUpdateSkills(computer.id);
  const { open: openRepair } = useFormDialog();
  const { open: confirm } = useConfirmationDialog();
  const [folded, setFolded] = useState(true);
  const [dialogOpen, setDialogOpen] = useState(!pairing && searchParams.get("setup") === computer.id);

  const days = daysUntil(computer.token_expires_at);
  const expired = !pairing && days <= 0;
  const expiringSoon = !expired && !pairing && days <= EXPIRY_WARNING_DAYS;
  const summary = setup && setupSummary(setup);
  const setupLabel = setup?.confirmed_at ? "Re-run setup" : "Set up";
  const primary: Primary = (() => {
    if (pairing) return "pair";
    if (expired) return "repair";
    if (setup && onlySkillsOutdated(setup)) return "update";
    if (summary?.state === "missing" || summary?.state === "failed") return "setup";
    if (expiringSoon) return "repair";
    return undefined;
  })();

  const repair = async () => {
    await openRepair<PairComputerFormData>({
      title: `Re-pair ${computer.name}`,
      description: "Get a fresh one-time token on the computer and paste it below.",
      schema: PairComputerFormSchema,
      okLabel: "Re-pair",
      form: <PairComputerForm computer={computer} />,
      formOptions: { defaultValues: { name: computer.name, server_url: computer.server_url, token: "" } },
    });
  };
  const removeComputer = async () => {
    const ok = await confirm({
      title: `Remove ${computer.name}?`,
      message: "@Agent stops working through this computer. Pair it again to bring it back.",
      confirmLabel: "Remove computer",
    });
    if (ok) remove.mutate(computer);
  };
  const forgetLink = () =>
    setSearchParams(
      (params) => {
        params.delete("setup");
        return params;
      },
      { replace: true },
    );

  const actions: RowAction[] = [
    ...(!pairing && primary !== "setup" ? [{ label: setupLabel, onSelect: () => setDialogOpen(true) }] : []),
    ...(!pairing && primary !== "repair" ? [{ label: "Re-pair", onSelect: () => void repair() }] : []),
    { label: "Remove", destructive: true, onSelect: () => void removeComputer() },
  ];
  const connection = connectionOf(presence);

  return (
    <li className="bg-card px-3 py-3">
      <div className="flex items-center gap-2">
        <button
          type="button"
          aria-expanded={!folded}
          onClick={() => setFolded(!folded)}
          className="group -ml-1 flex min-w-0 flex-1 items-center gap-1.5 rounded-md px-1 py-0.5 text-left focus-visible:outline-2 focus-visible:outline-focus"
        >
          <ChevronRight
            aria-hidden
            className={cn(
              "size-4 shrink-0 text-muted-foreground transition-transform duration-150 ease-standard group-hover:text-foreground",
              !folded && "rotate-90",
            )}
          />
          <span className="line-clamp-2 text-sm font-medium break-words" title={computer.name}>
            {computer.name}
          </span>
        </button>
        {primary === "pair" && (
          <Button type="button" variant="outline" size="sm" onClick={() => setDialogOpen(true)}>
            <RefreshCw className="size-4" aria-hidden />
            Pair
          </Button>
        )}
        {primary === "repair" && (
          <Button type="button" variant="outline" size="sm" onClick={() => void repair()}>
            <RefreshCw className="size-4" aria-hidden />
            Re-pair
          </Button>
        )}
        {primary === "update" && (
          <Button
            type="button"
            variant="outline"
            size="sm"
            loading={update.isPending}
            onClick={() => {
              update.mutate();
              setDialogOpen(true);
            }}
          >
            <RefreshCw className="size-4" aria-hidden />
            Update skills
          </Button>
        )}
        {primary === "setup" && (
          <Button type="button" variant="outline" size="sm" onClick={() => setDialogOpen(true)}>
            <Wrench className="size-4" aria-hidden />
            {setupLabel}
          </Button>
        )}
        <RowActionsMenu subject={computer.name} actions={actions} />
      </div>
      <p className="mt-1 flex min-w-0 flex-wrap items-center gap-x-3 gap-y-0.5 pl-6">
        {!pairing && <SettingsStatus tone={connection.tone}>{connection.text}</SettingsStatus>}
        {pairing && <SettingsStatus tone="warning">Pairing in progress</SettingsStatus>}
        {expired && <SettingsStatus tone="destructive">Pairing expired, acts as unpaired</SettingsStatus>}
        {expiringSoon && <SettingsStatus tone="warning">Pairing expires in {days}d</SettingsStatus>}
        {summary && (
          <SettingsStatus tone={summary.tone} detail={summary.detail}>
            {summary.text}
          </SettingsStatus>
        )}
      </p>
      <div inert={folded} aria-hidden={folded} data-closed={folded || undefined} className="disclosure">
        <div>
          <div className="pt-3 pl-6">
            <ComputerDetails computer={computer} setup={setup} />
          </div>
        </div>
      </div>
      {/* Mounted past pairing so the refetch that flips the row can't close the wizard as it reaches Set up. */}
      <PairComputerDialog existing={computer} open={dialogOpen} onOpenChange={setDialogOpen} onClosed={forgetLink} />
    </li>
  );
};
