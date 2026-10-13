import { EmptyRow } from "@/components/EmptyRow";
import { SettingsCard } from "@/components/settings/SettingsCard";
import { YourRolesRow } from "@/components/you/YourRolesRow";
import { useFetchWorkspaces } from "@/hooks/WorkspaceHooks";

// Read only: what you hold in each workspace, so a person can see why they see what they see.
export const YourRolesCard = () => {
  const { data: workspaces } = useFetchWorkspaces();
  return (
    <SettingsCard id="your-roles" title="Your roles" description="What you hold in each workspace. An Owner of that workspace changes them.">
      {workspaces && workspaces.length === 0 && <EmptyRow flush>You aren&apos;t in a workspace yet.</EmptyRow>}
      {workspaces && workspaces.length > 0 && (
        <ul className="divide-y divide-border border-t border-border">
          {workspaces.map((workspace) => (
            <YourRolesRow key={workspace.id} workspace={workspace} />
          ))}
        </ul>
      )}
    </SettingsCard>
  );
};
