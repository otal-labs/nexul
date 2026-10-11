import { useQueryClient } from "@tanstack/react-query";

import { playRenameImpactQuery } from "@/hooks/PlayHooks";
import { useConfirmationDialog } from "@/hooks/useConfirmationDialog";
import type { Play, PlayRenameImpact } from "@/models/Play";

const impactMessage = ({ count, automations }: PlayRenameImpact) => {
  const names = automations.length > 0 ? `: ${automations.join(", ")}` : "";
  if (count === 1) return `1 automation calls this play by its old name${names}. It'll fail until its code uses the new name.`;
  return `${count} automations call this play by its old name${names}. They'll fail until their code uses the new name.`;
};

// A rename is a signal, never a gate: automations still calling the old name with runPlay ask first, and a cancel is the only thing that stops it.
export const useConfirmPlayRename = (workspaceId: string) => {
  const client = useQueryClient();
  const { open: confirm } = useConfirmationDialog();
  return async (play: Play, label: string): Promise<boolean> => {
    if (label.trim().toLowerCase() === play.label.toLowerCase()) return true;
    const impact = await client.fetchQuery(playRenameImpactQuery(workspaceId, play.id, label));
    if (impact.count === 0) return true;
    return confirm({
      title: `Rename ${play.label}?`,
      message: impactMessage(impact),
      confirmLabel: "Rename anyway",
      destructive: false,
    });
  };
};
