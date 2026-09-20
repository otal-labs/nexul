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
        ? `releases ${hostnames.join(", ")} from the reverse proxy or Cloudflare tunnel`
        : "releases its hostnames from the reverse proxy or Cloudflare tunnel";
    const ok = await confirmDelete({
      message: `This removes the stack definition and its containers, and ${releaseClause}. Deploy history is kept for audit.`,
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
            Remove this stack definition and its containers, and release its hostnames from the reverse proxy
            or Cloudflare tunnel. Deploy history is kept for audit.
          </p>
          {hostnames.length > 0 && (
            <ul className="space-y-0.5">
              {hostnames.map((hostname) => (
                <li key={hostname} className="text-xs text-muted-foreground font-mono">
                  {hostname}
                </li>
              ))}
            </ul>
          )}
        </div>
        <Button
          variant="destructive"
          className="shrink-0"
          disabled={deleteStack.isPending}
          onClick={() => void onDelete()}
        >
          Delete stack
        </Button>
      </div>
    </SettingsCard>
  );
};
