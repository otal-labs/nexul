import { useState, type ReactNode } from "react";
import { Plus, RefreshCw, Wrench } from "lucide-react";
import { useSearchParams } from "react-router";

import { ComputerSettingsCard } from "@/components/computers/ComputerSettingsCard";
import { ComputerTerminalButton } from "@/components/computers/ComputerTerminalButton";
import { PaneHeader } from "@/components/computers/PaneHeader";
import { StaleComputerNote } from "@/components/computers/StaleComputerNote";
import { AddComputerDialog } from "@/components/pairing/AddComputerDialog";
import { SetupComputerDialog } from "@/components/pairing/SetupComputerDialog";
import { ComputerDetails } from "@/components/settings/ComputerDetails";
import { ComputerLoad } from "@/components/settings/ComputerLoad";
import { RenameComputerForm } from "@/components/settings/RenameComputerForm";
import { RowActionsMenu, type RowAction } from "@/components/settings/RowActionsMenu";
import { SettingsStatus, type SettingsStatusTone } from "@/components/settings/SettingsStatus";
import { Button } from "@/components/ui/button";
import { useFetchComputerSetup, useUpdateSkills } from "@/hooks/ComputerSetupHooks";
import { usePairNow, useRestartApp } from "@/hooks/ComputerHooks";
import { useDeleteComputer, useListOrderComputers } from "@/hooks/PairingHooks";
import { useComputerMoves } from "@/hooks/useComputerMoves";
import { useConfirmationDialog } from "@/hooks/useConfirmationDialog";
import { useFormDialog } from "@/hooks/useFormDialog";
import { usePermissionsDialog } from "@/hooks/usePermissionsDialog";
import { SETUP_PARAM } from "@/models/Agent";
import { computerLanes, isStaleOffline, offlineForDays, worstFirst } from "@/models/ComputerChecks";
import { computerEntity } from "@/models/PermissionPanel";
import { onlySkillsOutdated, RenameComputerFormSchema, setupSummary, stillPairing, type Computer, type RenameComputerFormData } from "@/models/Pairing";
import { daysUntil } from "@/utils/TimeUtility";

// Shown until the computer's app reports its hostname.
const UNNAMED = "New computer";

type Primary = "again" | "repair" | "update" | "setup" | undefined;

interface RunnerComputerPaneProps {
  computer: Computer;
  presence?: string | undefined;
}

// A computer added with its app: each lane on its own, at most one action for what needs doing, the rest in its menu, then
// its switches and settings and what is on it.
export const RunnerComputerPane = ({ computer, presence }: RunnerComputerPaneProps) => {
  const [searchParams, setSearchParams] = useSearchParams();
  const pairing = stillPairing(computer);
  const { data: setup } = useFetchComputerSetup(pairing ? "" : computer.id);
  const remove = useDeleteComputer();
  const update = useUpdateSkills(computer.id);
  const pairNow = usePairNow(computer);
  const restartApp = useRestartApp(computer);
  const { open: openRename } = useFormDialog();
  const { open: confirm } = useConfirmationDialog();
  const openPermissions = usePermissionsDialog();
  const moveActions = useComputerMoves(computer.id);
  const [setupOpen, setSetupOpen] = useState(!pairing && searchParams.get(SETUP_PARAM) === computer.id);

  const name = computer.name || UNNAMED;
  const connected = computer.runner?.connected === true;
  const { data: ordered } = useListOrderComputers();
  const rank = (ordered?.findIndex((c) => c.id === computer.id) ?? -1) + 1;
  const staleDays = isStaleOffline(computer) ? offlineForDays(computer) : 0;
  const summary = setup && setupSummary(setup);
  const lines: { tone: SettingsStatusTone; text: string; detail?: ReactNode; hollow?: boolean }[] = [
    ...computerLanes(computer, presence, pairNow.isPending),
    ...(summary ? [{ tone: summary.tone, text: summary.text }] : []),
  ];
  const setupLabel = setup?.confirmed_at ? "Re-run setup" : "Set up";
  const primary: Primary = (() => {
    if (!computer.runner) return "again";
    if (connected && (computer.pair_error || presence === "rejected" || (!pairing && daysUntil(computer.token_expires_at) <= 0))) return "repair";
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
        params.delete(SETUP_PARAM);
        return params;
      },
      { replace: true },
    );

  const actions: RowAction[] = [
    ...(!pairing && primary !== "setup" ? [{ label: setupLabel, onSelect: () => setSetupOpen(true) }] : []),
    ...(connected && primary !== "repair" ? [{ label: "Re-pair now", disabled: pairNow.isPending, onSelect: () => pairNow.mutate() }] : []),
    ...(connected ? [{ label: "Restart Nexul app", disabled: restartApp.isPending, onSelect: () => void restartApp.restart() }] : []),
    { label: "Share…", onSelect: () => void openPermissions(computerEntity({ id: computer.id, name })) },
    { label: "Rename", onSelect: () => void rename() },
    ...moveActions,
    { label: "Remove", destructive: true, onSelect: () => void removeComputer() },
  ];

  return (
    <div className="space-y-6">
      <PaneHeader
        title={name}
        mono
        note="Changes save as you go"
        meta={
          <>
            {worstFirst(lines).map((line) => (
              <SettingsStatus
                key={line.text}
                tone={line.tone}
                hollow={!!line.hollow}
                {...(line.tone === "warning" && line.hollow && { textClassName: "text-warning" })}
                detail={line.hollow ? undefined : line.detail}
              >
                {line.hollow && line.detail ? `${line.text} · ${line.detail}` : line.text}
              </SettingsStatus>
            ))}
            {!pairing && <ComputerLoad computer={computer} />}
          </>
        }
        actions={
          <>
            {computer.shell_enabled && <ComputerTerminalButton computer={computer} offline={!connected} />}
            {/* Mounted whatever the pane shows, so the pane changing as the computer connects can't close the dialog. */}
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
          </>
        }
      />
      {staleDays > 0 && <StaleComputerNote rank={rank} onRemove={() => void removeComputer()} />}
      {!pairing && <ComputerSettingsCard computer={computer} setup={setup} />}
      <ComputerDetails computer={computer} setup={setup} />
      {!pairing && <SetupComputerDialog computer={computer} open={setupOpen} onOpenChange={setSetupOpen} onClosed={forgetLink} />}
    </div>
  );
};
