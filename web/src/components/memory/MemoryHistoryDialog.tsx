import { HistoryDialog } from "@/components/history/HistoryDialog";
import { memoryHistoryUrl, memoryVersionUrl, useFetchMemory } from "@/hooks/MemoryHooks";
import { useRestoreMemoryVersion } from "@/hooks/MemoryHistoryHooks";

interface MemoryHistoryDialogProps {
  memoryId: string;
  /** The version the dialog opens on. */
  version: number;
  onClose: () => void;
}

export const MemoryHistoryDialog = ({ memoryId, version, onClose }: MemoryHistoryDialogProps) => {
  const memory = useFetchMemory(memoryId).data;
  const restore = useRestoreMemoryVersion(memoryId);

  return (
    <HistoryDialog
      historyUrl={memoryHistoryUrl(memoryId)}
      versionUrl={(n) => memoryVersionUrl(memoryId, n)}
      name={memory?.title ?? ""}
      initialVersion={version}
      retention="ten"
      restore={{
        permission: "memories:write",
        projectId: memory?.project_id,
        answer: undefined,
        run: (n) => restore.mutate(n, { onSuccess: onClose }),
        pending: restore.isPending,
      }}
      onClose={onClose}
    />
  );
};
