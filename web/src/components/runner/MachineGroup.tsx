import { DownloadIcon, ServerIcon } from "lucide-react";
import { machineAnswer } from "@nexul/client-core/actionState";
import { hasPermission } from "@nexul/client-core/permissions";

import { ActionButton } from "@/components/action/ActionButton";
import { EnterList } from "@/components/EnterList";
import { EmptyRow } from "@/components/EmptyRow";
import { RowActions } from "@/components/listpane/RowActions";
import { RowContextMenu } from "@/components/listpane/RowContextMenu";
import { offersMenu, type RowMenu } from "@/components/listpane/rowMenu";
import { AddRunnerDialog } from "@/components/runner/AddRunnerDialog";
import { EditableStackRoot } from "@/components/runner/EditableStackRoot";
import { MachineDeleteDetails } from "@/components/runner/MachineDeleteDetails";
import { MachinePeopleRow } from "@/components/runner/MachinePeopleRow";
import { RunnerRow } from "@/components/runner/RunnerRow";
import { useActionState } from "@/hooks/useActionState";
import { useDeleteMachine } from "@/hooks/MachineHooks";
import { useConfirmationDialog } from "@/hooks/useConfirmationDialog";
import { usePermissionsDialog } from "@/hooks/usePermissionsDialog";
import { useWorkspacePath } from "@/hooks/useWorkspacePath";
import { machineEntity } from "@/models/PermissionPanel";
import type { Machine, Runner } from "@/models/Runner";
import { cn } from "@/lib/utils";
import { formatRelativeTime } from "@/utils/TimeUtility";

interface MachineGroupProps {
  machine: Machine;
  runners: Runner[];
}

type Tone = "online" | "partial" | "offline";

const toneOf = (online: number, total: number): Tone => {
  if (total > 0 && online === total) return "online";
  if (online > 0) return "partial";
  return "offline";
};

// Each fact carries its own leading dot; the list is pulled left under a clip, so a dot that would start a line is cut off.
const factClass = "relative min-w-0 pl-4 wrap-anywhere before:absolute before:left-[0.3rem] before:text-muted-foreground/40 before:content-['·']";

// A machine with nothing connected loses its lift and draws a dashed edge, like an unplugged card; its text keeps full contrast.
export const MachineGroup = ({ machine, runners }: MachineGroupProps) => {
  const wsPath = useWorkspacePath();
  const openPermissions = usePermissionsDialog();
  const manages = hasPermission(machine.actions, "machines:write");
  const deleteGate = { permission: "machines:delete", answer: machineAnswer(machine, "machines:delete") } as const;
  const canDelete = useActionState(deleteGate).state !== "hidden";
  const deleteMachine = useDeleteMachine();
  const { open: confirmDelete } = useConfirmationDialog();
  const entity = machineEntity(machine);
  const onDelete = async () => {
    const ok = await confirmDelete({
      title: `Delete machine ${machine.name}?`,
      message:
        "Adding a runner on it again brings the machine back, without its rules. It is refused while a stack or gateway uses it or a runner on it is connected.",
      details: <MachineDeleteDetails machine={machine} runners={runners} />,
      confirmLabel: "Delete machine",
    });
    if (ok) deleteMachine.mutate(machine.id);
  };
  const menu: RowMenu = {
    permissions: manages ? { entity, onOpen: () => void openPermissions(entity) } : undefined,
    onDelete: canDelete ? () => void onDelete() : undefined,
    deleteGate,
  };
  const online = runners.filter((r) => r.connected).length;
  const tone = toneOf(online, runners.length);
  const status = tone === "partial" ? `${online} of ${runners.length} online` : tone;
  return (
    <RowContextMenu menu={menu} manages={manages}>
      <section
        aria-labelledby={`machine-${machine.id}`}
        className={cn(
          "@container overflow-hidden rounded-lg border border-border",
          tone === "offline" ? "border-dashed" : "bg-card shadow-card",
        )}
      >
        <header className="grid grid-cols-[auto_minmax(0,1fr)_auto] items-center gap-x-3 gap-y-1 px-4 py-3.5">
          <span className="relative row-span-2 flex size-9 shrink-0 items-center justify-center self-start rounded-md border border-border bg-surface-2">
            <ServerIcon className={cn("size-4", tone === "offline" && "text-muted-foreground")} aria-hidden />
            <span
              aria-hidden
              className={cn(
                "absolute -top-1 -right-1 size-2.5 rounded-full ring-2 ring-card",
                tone === "online" && "bg-success",
                tone === "partial" && "bg-warning",
                tone === "offline" && "bg-muted-foreground/60",
              )}
            />
          </span>
          <div className="flex flex-wrap items-baseline gap-x-2.5 gap-y-0.5">
            <h2 id={`machine-${machine.id}`} className="min-w-0 text-sm font-semibold wrap-anywhere">
              {machine.name}
            </h2>
            <span className={cn("text-xs", tone === "online" ? "text-success" : "text-muted-foreground")}>{status}</span>
          </div>
          <div className="-my-1 -mr-1.5 flex items-center">
            <ActionButton
              permission={["machines:write", "stacks:write"]}
              answer={machineAnswer(machine, "machines:write")}
              verb="import from machines"
              variant="ghost"
              size="sm"
              icon={DownloadIcon}
              aria-label="Import from this machine"
              title="Import from this machine"
              to={wsPath(`/wizard/project/import?machine=${machine.id}`)}
            >
              <span className="hidden @lg:inline">Import</span>
            </ActionButton>
            <AddRunnerDialog machine={machine} triggerSize="sm" triggerVariant="ghost" />
            {offersMenu(menu, manages) && <RowActions itemLabel={machine.name} {...menu} />}
          </div>
          <div className="col-span-2 overflow-hidden">
            <div className="-ml-4 flex flex-wrap gap-y-1 font-mono text-xs text-muted-foreground">
              {machine.reported_hostname && machine.reported_hostname !== machine.name && (
                <span className={factClass}>{machine.reported_hostname}</span>
              )}
              <span className={factClass}>
                {runners.length} runner{runners.length === 1 ? "" : "s"}
              </span>
              <span className={factClass}>seen {formatRelativeTime(machine.last_seen)}</span>
              <span className={factClass}>
                <EditableStackRoot machine={machine} />
              </span>
            </div>
          </div>
        </header>
        {manages && <MachinePeopleRow machine={machine} />}
        {runners.length === 0 && <EmptyRow className="border-t border-border">No runners on this machine yet.</EmptyRow>}
        {runners.length > 0 && (
          <EnterList className="divide-y divide-border border-t border-border">
            {runners.map((runner) => (
              <RunnerRow key={runner.id} runner={runner} machine={machine} />
            ))}
          </EnterList>
        )}
      </section>
    </RowContextMenu>
  );
};
