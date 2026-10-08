import { useNavigate } from "react-router";

import { useDeleteMemory } from "@/hooks/MemoryHooks";
import { useConfirmationDialog } from "@/hooks/useConfirmationDialog";
import { useWorkspacePath } from "@/hooks/useWorkspacePath";
import type { Memory } from "@/models/Memory";

// Asks first; leaveAfter returns to the list when the deleted memory is the one open.
export const useConfirmDeleteMemory = () => {
  const navigate = useNavigate();
const wsPath = useWorkspacePath();
  const { open: confirm } = useConfirmationDialog();
  const deleteMemory = useDeleteMemory();

  return async (memory: Memory, leaveAfter: boolean) => {
    const ok = await confirm({
      title: "Delete memory?",
      message: `"${memory.title}" and its versions are deleted for good.`,
      confirmLabel: "Delete memory",
    });
    if (!ok) return;
    deleteMemory.mutate(memory.id, {
      onSuccess: () => {
        if (leaveAfter) void navigate(wsPath("/memories"));
      },
    });
  };
};
