import { useNavigate } from "react-router";

import { DangerAction, DangerButton, DangerZone } from "@/components/settings/DangerZone";
import { useDeleteStack } from "@/hooks/StackHooks";
import { useConfirmationDialog } from "@/hooks/useConfirmationDialog";
import type { Stack } from "@/models/Stack";

interface StackDangerZoneSectionProps {
  stack: Stack;
  projectPath: string;
  hostnames: string[];
}

export const StackDangerZoneSection = ({ stack, projectPath, hostnames }: StackDangerZoneSectionProps) => {
  const navigate = useNavigate();
  const deleteStack = useDeleteStack();
  const { open: confirmDelete } = useConfirmationDialog();

  const onDelete = async () => {
    const releaseClause =
      hostnames.length > 0
        ? `${hostnames.join(", ")} stop routing to it`
        : "its hostnames stop routing to it";
    const ok = await confirmDelete({
      message: `The stack and its containers are removed, and ${releaseClause}. Deploy history is kept.`,
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
        consequence="Removes the stack and its containers. Its hostnames stop routing to it. Deploy history is kept."
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
          <DangerButton loading={deleteStack.isPending} onClick={() => void onDelete()}>
            Delete stack
          </DangerButton>
        }
      />
    </DangerZone>
  );
};
