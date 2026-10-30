import type { ReactNode } from "react";
import { machineAnswer } from "@nexul/client-core/actionState";
import { hasPermission } from "@nexul/client-core/permissions";

import { RowContextMenu } from "@/components/listpane/RowContextMenu";
import type { RowMenu } from "@/components/listpane/rowMenu";
import { AddRunnerDialog } from "@/components/runner/AddRunnerDialog";
import { GroupDot } from "@/components/runner/GroupDot";
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
import { groupState } from "@/utils/RunnerFleetUtility";

interface MachineBandProps {
  machine: Machine;
  runners: Runner[];
  codes: number;
  toggle?: ReactNode;
}

// The group's head, pinned while its rows scroll: the dot sits on the runner dots' x and the name keeps its full text.
export const MachineBand = ({ machine, runners, codes, toggle }: MachineBandProps) => {
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
  const state = groupState(runners, codes, machine.last_seen);
  const host = machine.reported_hostname && machine.reported_hostname !== machine.name ? machine.reported_hostname : "";

  return (
    <RowContextMenu menu={menu} manages={manages}>
      <div className="sticky top-0 z-10 flex items-start gap-3 border-b border-border group-has-[>ul[hidden]]/fleet:border-b-0 bg-card bg-linear-to-b from-well-shade to-well-shade px-4 py-2.5">
        <GroupDot tone={state.tone} />
        <div className="min-w-0 flex-1">
          <div className="flex flex-wrap items-baseline gap-x-2.5">
            <h2 id={`machine-${machine.id}`} className="min-w-0 text-sm/5 font-semibold wrap-anywhere">
              {machine.name}
            </h2>
            <span className="text-xs text-muted-foreground tabular-nums">{state.words}</span>
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
          {toggle}
          {manages && <MachinePeopleButton machine={machine} className="max-w-32 @[44rem]:max-w-56" />}
          {runners.length + codes > 0 && <AddRunnerDialog machine={machine} triggerSize="sm" triggerVariant="ghost" compact />}
          <MachineMenu machine={machine} menu={menu} manages={manages} />
        </div>
      </div>
    </RowContextMenu>
  );
};
