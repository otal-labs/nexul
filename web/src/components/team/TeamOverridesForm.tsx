import { PermissionGrid } from "@/components/access/PermissionGrid";
import { PermissionLevels } from "@/components/access/PermissionLevels";
import { ErrorDisplay } from "@/components/ErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { useFetchPermissionCatalog } from "@/hooks/PermissionHooks";
import { useMemberDraft } from "@/hooks/useMemberDraft";
import type { TeamMembership } from "@/models/Team";

interface TeamOverridesFormProps {
  workspaceId: string;
  membership: TeamMembership;
}

// The same allow levels and deny grid an invitation uses; changes join the dialog's draft like everything else.
export const TeamOverridesForm = ({ workspaceId, membership }: TeamOverridesFormProps) => {
  const { data: catalog, isPending, error } = useFetchPermissionCatalog();
  const { dispatch } = useMemberDraft();
  const overlap = membership.allow.some((value) => membership.deny.includes(value));

  return (
    <div className="space-y-3">
      {isPending && <LoadingDisplay />}
      {error && <ErrorDisplay error={error} />}
      {catalog && (
        <div className="space-y-1">
          <p className="text-xs font-medium">Allow</p>
          <PermissionLevels entries={catalog} value={membership.allow} onChange={(allow) => dispatch({ type: "change", workspaceId, change: { allow } })} />
        </div>
      )}
      {catalog && (
        <div className="space-y-1">
          <p className="text-xs font-medium">Deny</p>
          <PermissionGrid entries={catalog} value={membership.deny} onChange={(deny) => dispatch({ type: "change", workspaceId, change: { deny } })} />
        </div>
      )}
      {overlap && <p role="alert" className="text-sm text-destructive">A permission can&apos;t be allowed and denied at once.</p>}
    </div>
  );
};
