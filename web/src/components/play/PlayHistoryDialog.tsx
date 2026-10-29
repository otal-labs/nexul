import { HistoryDialog } from "@/components/history/HistoryDialog";
import { useFetchProjects } from "@/hooks/ProjectHooks";
import { playHistoryUrl, playVersionUrl, useRestorePlayVersion } from "@/hooks/PlayHistoryHooks";
import { useEntityActions } from "@/hooks/useEntityActions";
import { readPlayChange } from "@/utils/PlayHistoryUtility";

interface PlayHistoryDialogProps {
  playId: string;
  name: string;
  /** The version to open on, instead of the newest. */
  version?: number;
  onClose: () => void;
}

// A play's instructions are markdown text, so the dialog cuts them at blank lines; its other fields list as rows beside the versions.
export const PlayHistoryDialog = ({ playId, name, version, onClose }: PlayHistoryDialogProps) => {
  const answer = useEntityActions("play", playId);
  const restore = useRestorePlayVersion(playId);
  const { data: projects } = useFetchProjects();

  return (
    <HistoryDialog
      historyUrl={playHistoryUrl(playId)}
      versionUrl={(n) => playVersionUrl(playId, n)}
      name={name}
      bodyKind="markdown"
      {...(version !== undefined && { initialVersion: version })}
      readChange={readPlayChange((id) => projects?.find((project) => project.id === id)?.name ?? "A removed project")}
      restore={{
        permission: "plays:write",
        answer: answer("plays:write"),
        run: (n) => restore.mutate(n, { onSuccess: onClose }),
        pending: restore.isPending,
      }}
      onClose={onClose}
    />
  );
};
