import { useState } from "react";
import { ChevronRight, Plus, RefreshCw, Wrench } from "lucide-react";
import { useSearchParams } from "react-router";

import { AddComputerDialog } from "@/components/pairing/AddComputerDialog";
import { PairComputerDialog } from "@/components/pairing/PairComputerDialog";
import { ComputerDetails } from "@/components/settings/ComputerDetails";
import { RenameComputerForm } from "@/components/settings/RenameComputerForm";
import { RowActionsMenu, type RowAction } from "@/components/settings/RowActionsMenu";
import { SettingsStatus } from "@/components/settings/SettingsStatus";
import { Button } from "@/components/ui/button";
import { useFetchComputerSetup, useUpdateSkills } from "@/hooks/ComputerSetupHooks";
import { usePairNow } from "@/hooks/ComputerHooks";
import { useDeleteComputer } from "@/hooks/PairingHooks";
import { useConfirmationDialog } from "@/hooks/useConfirmationDialog";
import { useFormDialog } from "@/hooks/useFormDialog";
import { computerLanes } from "@/models/ComputerChecks";
import { onlySkillsOutdated, RenameComputerFormSchema, setupSummary, stillPairing, type Computer, type RenameComputerFormData } from "@/models/Pairing";
import { cn } from "@/lib/utils";
import { daysUntil } from "@/utils/TimeUtility";

// Shown until the computer's app reports its hostname.
const UNNAMED = "New computer";

type Primary = "again" | "repair" | "update" | "setup" | undefined;

interface RunnerComputerRowProps {
  computer: Computer;
  presence?: string | undefined;
}

// A computer added with its app: each lane's health on its own, at most one action for what needs doing, the rest in its menu.
export const RunnerComputerRow = ({ computer, presence }: RunnerComputerRowProps) => {
  const [searchParams, setSearchParams] = useSearchParams();
  const pairing = stillPairing(computer);
  const { data: setup } = useFetchComputerSetup(pairing ? "" : computer.id);
  const remove = useDeleteComputer();
  const update = useUpdateSkills(computer.id);
  const pairNow = usePairNow(computer);
  const { open: openRename } = useFormDialog();
  const { open: confirm } = useConfirmationDialog();
  const [folded, setFolded] = useState(true);
  const [setupOpen, setSetupOpen] = useState(!pairing && searchParams.get("setup") === computer.id);

  const name = computer.name || UNNAMED;
  const connected = computer.runner?.connected === true;
  const summary = setup && setupSummary(setup);
  const setupLabel = setup?.confirmed_at ? "Re-run setup" : "Set up";
  const primary: Primary = (() => {
    if (!computer.runner) return "again";
    if (connected && (computer.pair_error || (!pairing && daysUntil(computer.token_expires_at) <= 0))) return "repair";
    if (setup && onlySkillsOutdated(setup)) return "update";
    if (summary?.state === "missing" || summary?.state === "failed") return "setup";
    return undefined;
  })();

  const rename = () =>
    openRename<RenameComputerFormData>({
      title: `Rename ${name}`,
      schema: RenameComputerFormSchema,
      okLabel: "Rename",
      form: <RenameComputerForm computerId={computer.id} />,
      formOptions: { defaultValues: { name: computer.name } },
    });
  const removeComputer = async () => {
    const ok = await confirm({
      title: `Remove ${name}?`,
      message: "@Agent stops working through this computer, and the Nexul app removes itself from it. Add it again to bring it back.",
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
    ...(!pairing && primary !== "setup" ? [{ label: setupLabel, onSelect: () => setSetupOpen(true) }] : []),
    ...(connected && primary !== "repair" ? [{ label: "Re-pair now", onSelect: () => pairNow.mutate() }] : []),
    { label: "Rename", onSelect: () => void rename() },
    { label: "Remove", destructive: true, onSelect: () => void removeComputer() },
  ];

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
            className={cn("size-4 shrink-0 text-muted-foreground transition-transform duration-150 ease-standard group-hover:text-foreground", !folded && "rotate-90")}
          />
          <span className={cn("line-clamp-2 text-sm font-medium break-words", !computer.name && "text-muted-foreground")} title={name}>
            {name}
          </span>
        </button>
        {/* Mounted whatever the row shows, so the row changing as the computer connects can't close the dialog. */}
        <AddComputerDialog
          again={{ id: computer.id, name: computer.name }}
          trigger={
            primary === "again" && (
              <Button type="button" variant="outline" size="sm">
                <Plus className="size-4" aria-hidden />
                Add this computer again
              </Button>
            )
          }
        />
        {primary === "repair" && (
          <Button type="button" variant="outline" size="sm" loading={pairNow.isPending} onClick={() => pairNow.mutate()}>
            <RefreshCw className="size-4" aria-hidden />
            Re-pair now
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
              setSetupOpen(true);
            }}
          >
            <RefreshCw className="size-4" aria-hidden />
            Update skills
          </Button>
        )}
        {primary === "setup" && (
          <Button type="button" variant="outline" size="sm" onClick={() => setSetupOpen(true)}>
            <Wrench className="size-4" aria-hidden />
            {setupLabel}
          </Button>
        )}
        <RowActionsMenu subject={name} actions={actions} />
      </div>
      <p className="mt-1 flex min-w-0 flex-wrap items-center gap-x-3 gap-y-0.5 pl-6">
        {computerLanes(computer, presence).map((lane) => (
          <SettingsStatus key={lane.text} tone={lane.tone} detail={lane.detail}>
            {lane.text}
          </SettingsStatus>
        ))}
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
      {!pairing && <PairComputerDialog existing={computer} open={setupOpen} onOpenChange={setSetupOpen} onClosed={forgetLink} />}
    </li>
  );
};
