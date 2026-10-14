import { ArchiveIcon } from "lucide-react";

import { EmptyRow } from "@/components/EmptyRow";
import { ErrorDisplay } from "@/components/ErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { ArchivedWorkspaceRow } from "@/components/settings/ArchivedWorkspaceRow";
import { SettingsCard } from "@/components/settings/SettingsCard";
import { useHasInstancePermission } from "@/hooks/AccessHooks";
import { useFetchArchivedWorkspaces } from "@/hooks/WorkspaceHooks";

const ArchivedWorkspacesCard = () => {
  const { data: workspaces, error, isPending } = useFetchArchivedWorkspaces(true);
  return (
    <SettingsCard
      id="archived-workspaces"
      title="Archived workspaces"
      description="An archived workspace is hidden from everyone in it. Restoring one brings back its projects, people and roles as they were."
      icon={ArchiveIcon}
    >
      {isPending && <LoadingDisplay />}
      {error && <ErrorDisplay error={error} />}
      {workspaces && workspaces.length === 0 && <EmptyRow flush>No workspace is archived.</EmptyRow>}
      {workspaces && workspaces.length > 0 && (
        <ul className="-mx-3 divide-y divide-border">
          {workspaces.map((workspace) => (
            <ArchivedWorkspaceRow key={workspace.id} workspace={workspace} />
          ))}
        </ul>
      )}
    </SettingsCard>
  );
};

// Restoring takes instance:write, which only the instance Owner holds by right.
export const ArchivedWorkspacesSection = () => {
  const canRestore = useHasInstancePermission("instance:write");
  return <>{canRestore && <ArchivedWorkspacesCard />}</>;
};
