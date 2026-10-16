import { useNavigate } from "react-router";

import { ActionButton } from "@/components/action/ActionButton";
import { DangerAction, dangerButtonClass, DangerZone } from "@/components/settings/DangerZone";
import { useDeleteStack } from "@/hooks/StackHooks";
import { useConfirmationDialog } from "@/hooks/useConfirmationDialog";
import { useStackActions } from "@/hooks/useStackActions";
import type { Stack } from "@/models/Stack";

interface StackDangerZoneSectionProps {
  stack: Stack;
  projectPath: string;
  hostnames: string[];
}

// A managed stack's runner removes its containers once it is online; an adopted stack's were started outside Nexul.
const containersClause = (managed: boolean) => {
  if (managed) return "its machine's runner removes its containers, once it is online if it is not now";
  return "its containers keep running, because they were started outside Nexul";
};

export const StackDangerZoneSection = ({ stack, projectPath, hostnames }: StackDangerZoneSectionProps) => {
  const navigate = useNavigate();
  const deleteStack = useDeleteStack();
  const { open: confirmDelete } = useConfirmationDialog();
  const answer = useStackActions(stack);

  const onDelete = async () => {
    const releaseClause =
      hostnames.length > 0
        ? `${hostnames.join(", ")} stop routing to it`
        : "its hostnames stop routing to it";
    const ok = await confirmDelete({
      message: `The stack is deleted and ${containersClause(stack.managed)}. Volumes stay on the machine, and ${releaseClause}. Deploy history is kept.`,
      title: `Delete ${stack.name}?`,
      confirmLabel: "Delete stack",
    });
    if (!ok) return;
    await deleteStack.mutateAsync(stack.id);
    navigate(projectPath);
  };

  return (
    <DangerZone>
      <DangerAction
        title="Delete stack"
        consequence={`Deletes the stack; ${containersClause(stack.managed)}. Its hostnames stop routing to it. Deploy history is kept.`}
        details={
          hostnames.length > 0 && (
            <ul className="space-y-0.5 pt-1">
              {hostnames.map((hostname) => (
                <li key={hostname} className="font-mono text-xs wrap-anywhere text-muted-foreground">
                  {hostname}
                </li>
              ))}
            </ul>
          )
        }
        action={
          <ActionButton
            permission="stacks:delete"
            projectId={stack.project_id || undefined}
            answer={answer("stacks:delete")}
            variant="outline"
            size="sm"
            liveClassName={dangerButtonClass}
            loading={deleteStack.isPending}
            onClick={() => void onDelete()}
          >
            Delete stack
          </ActionButton>
        }
      />
    </DangerZone>
  );
};
