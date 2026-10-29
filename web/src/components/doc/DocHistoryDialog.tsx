import { HistoryDialog } from "@/components/history/HistoryDialog";
import { docHistoryUrl, docVersionUrl, useRestoreDocVersion } from "@/hooks/DocHistoryHooks";
import { useEntityActions } from "@/hooks/useEntityActions";
import type { Doc } from "@/models/Doc";

interface DocHistoryDialogProps {
  doc: Doc;
  onClose: () => void;
}

export const DocHistoryDialog = ({ doc, onClose }: DocHistoryDialogProps) => {
  const answer = useEntityActions("doc", doc.id);
  const restore = useRestoreDocVersion(doc.id);

  return (
    <HistoryDialog
      historyUrl={docHistoryUrl(doc.id)}
      versionUrl={(version) => docVersionUrl(doc.id, version)}
      name={doc.title}
      restore={{
        permission: "docs:write",
        projectId: doc.project_id,
        answer: answer("docs:write"),
        run: (version) => restore.mutate(version, { onSuccess: onClose }),
        pending: restore.isPending,
      }}
      onClose={onClose}
    />
  );
};
