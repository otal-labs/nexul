import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { useAssignInstallation } from "@/hooks/RepositoryHooks";
import { useFetchWorkspaces } from "@/hooks/WorkspaceHooks";
import type { Installation } from "@/models/Repository";

interface GitHubInstallationAssignProps {
  installation: Installation;
}

// Picks a workspace of the caller's to list this installation's repositories too; hidden once every one does.
export const GitHubInstallationAssign = ({ installation }: GitHubInstallationAssignProps) => {
  const { data: workspaces } = useFetchWorkspaces();
  const assign = useAssignInstallation();
  const open = workspaces?.filter((w) => !installation.workspaces.some((a) => a.id === w.id)) ?? [];
  if (open.length === 0) return null;

  return (
    <Select
      value=""
      disabled={assign.isPending}
      onValueChange={(workspaceId) => assign.mutate({ account: installation.account_login, workspaceId })}
    >
      <SelectTrigger
        aria-label={`Assign ${installation.account_login} to a workspace`}
        className="h-6 w-auto gap-1 rounded-full border-dashed px-2 text-xs"
      >
        <SelectValue placeholder="Assign to…" />
      </SelectTrigger>
      <SelectContent>
        {open.map((w) => (
          <SelectItem key={w.id} value={w.id}>
            {w.name}
          </SelectItem>
        ))}
      </SelectContent>
    </Select>
  );
};
