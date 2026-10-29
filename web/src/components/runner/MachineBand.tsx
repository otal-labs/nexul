import { machineAnswer } from "@nexul/client-core/actionState";
import { hasPermission } from "@nexul/client-core/permissions";

import { RowContextMenu } from "@/components/listpane/RowContextMenu";
import type { RowMenu } from "@/components/listpane/rowMenu";
import { AddRunnerDialog } from "@/components/runner/AddRunnerDialog";
import { EditableStackRoot } from "@/components/runner/EditableStackRoot";
import { MachineDeleteDetails } from "@/components/runner/MachineDeleteDetails";
import { MachineMenu } from "@/components/runner/MachineMenu";
import { MachinePeopleButton } from "@/components/runner/MachinePeopleButton";
import { useActionState } from "@/hooks/useActionState";
import { useDeleteMachine } from "@/hooks/MachineHooks";
import { useConfirmationDialog } from "@/hooks/useConfirmationDialog";
import { usePermissionsDialog } from "@/hooks/usePermissionsDialog";
import { machineEntity } from "@/models/PermissionPanel";
import type { Machine, Runner } from "@/models/Runner";
import { cn } from "@/lib/utils";
import { formatRelativeTime } from "@/utils/TimeUtility";

interface MachineBandProps {
  machine: Machine;
  runners: Runner[];
}

type Tone = "online" | "partial" | "offline";

const toneOf = (online: number, total: number): Tone => {
  if (total > 0 && online === total) return "online";
  if (online > 0) return "partial";
  return "offline";
};

const stateOf = (tone: Tone, online: number, total: number, machine: Machine) => {
  if (tone === "partial") return `${online} of ${total} online`;
  if (tone === "online") return "online";
  return `offline · seen ${formatRelativeTime(machine.last_seen)}`;
};

// The group's head: the dot sits on the runner dots' x, the name keeps its full text, and only + and ⋯ trail below 44rem.
export const MachineBand = ({ machine, runners }: MachineBandProps) => {
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
  const host = machine.reported_hostname && machine.reported_hostname !== machine.name ? machine.reported_hostname : "";

  return (
    <RowContextMenu menu={menu} manages={manages}>
      <div className="flex items-start gap-3 border-b border-border bg-well-shade px-4 py-2.5">
        <span className="flex h-5 shrink-0 items-center" title={tone}>
          <span
            aria-hidden
            className={cn(
              "size-2 rounded-full",
              tone === "online" && "bg-success",
              tone === "partial" && "bg-warning",
              tone === "offline" && "ring-[1.5px] ring-muted-foreground/60 ring-inset",
            )}
          />
        </span>
        <div className="min-w-0 flex-1">
          <div className="flex flex-wrap items-baseline gap-x-2.5">
            <h2 id={`machine-${machine.id}`} className="min-w-0 text-sm/5 font-semibold wrap-anywhere">
              {machine.name}
            </h2>
            <span className="text-xs text-muted-foreground">{stateOf(tone, online, runners.length, machine)}</span>
          </div>
          <div className="flex min-w-0 items-baseline gap-2 font-mono text-xs text-muted-foreground">
            {host && <span className="min-w-0 truncate">{host}</span>}
            {host && <span aria-hidden className="font-sans text-muted-foreground/40">·</span>}
            <span className="min-w-0 truncate">
              <EditableStackRoot machine={machine} />
            </span>
          </div>
        </div>
        <div className="-my-1.5 flex shrink-0 items-center gap-0.5">
          {manages && <MachinePeopleButton machine={machine} className="hidden @[44rem]:inline-flex" />}
          <AddRunnerDialog machine={machine} triggerSize="sm" triggerVariant="ghost" iconOnly />
          <MachineMenu machine={machine} menu={menu} manages={manages} />
        </div>
      </div>
    </RowContextMenu>
  );
};
