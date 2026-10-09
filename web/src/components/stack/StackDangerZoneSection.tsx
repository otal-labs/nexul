import { TriangleAlert } from "lucide-react";
import { useNavigate } from "react-router";

import { SettingsCard } from "@/components/settings/SettingsCard";
import { Button } from "@/components/ui/button";
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
      confirmLabel: "Delete",
    });
    if (!ok) return;
    await deleteStack.mutateAsync(stack.id);
    navigate(projectPath);
  };

  return (
    <SettingsCard id="danger-zone" title="Danger zone" danger icon={TriangleAlert}>
      <div className="flex flex-col items-start justify-between gap-4 sm:flex-row sm:items-center">
        <div className="space-y-2">
          <p className="text-sm text-muted-foreground">
            Deleting removes the stack and its containers, and its hostnames stop routing to it. Deploy history is kept.
          </p>
          {hostnames.length > 0 && (
            <ul className="space-y-0.5">
              {hostnames.map((hostname) => (
                <li key={hostname} className="font-mono text-xs wrap-anywhere text-muted-foreground">
                  {hostname}
                </li>
              ))}
            </ul>
          )}
        </div>
        <Button
          variant="destructive"
          className="shrink-0"
          loading={deleteStack.isPending}
          onClick={() => void onDelete()}
        >
          Delete stack
        </Button>
      </div>
    </SettingsCard>
  );
};
