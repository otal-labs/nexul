import { ErrorDisplay } from "@/components/ErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { CreateInvitationDialog } from "@/components/member/CreateInvitationDialog";
import { InvitationsFeed } from "@/components/member/InvitationsFeed";
import { MembersFeed } from "@/components/member/MembersFeed";
import { NoDataDisplay } from "@/components/NoDataDisplay";
import { SettingsCard } from "@/components/settings/SettingsCard";
import {
  useChangeWorkspaceMemberRole,
  useFetchWorkspaceMembers,
  useRemoveWorkspaceMember,
} from "@/hooks/MemberHooks";
import { useFetchWorkspaceRoles } from "@/hooks/RoleHooks";
import { useWorkspaceStore } from "@/stores/workspaceStore";

export const MembersSection = () => {
  const selectedWorkspaceId = useWorkspaceStore((s) => s.selectedWorkspaceId);
  const { data: members, isPending, error } = useFetchWorkspaceMembers(selectedWorkspaceId);
  const { data: roles } = useFetchWorkspaceRoles(selectedWorkspaceId);
  const removeMember = useRemoveWorkspaceMember(selectedWorkspaceId);
  const changeRole = useChangeWorkspaceMemberRole(selectedWorkspaceId);

  return (
    <SettingsCard
      id="members"
      title="Members"
      description="Who is in this workspace and with which role, plus private invitation links with access across one or more workspaces."
      footer={<CreateInvitationDialog />}
    >
      {isPending && <LoadingDisplay />}
      {error && <ErrorDisplay error={error} />}
      {members && members.members.length === 0 && (
        <NoDataDisplay message="No members yet — create an invitation link to add someone." />
      )}
      {members && members.members.length > 0 && (
        <MembersFeed
          members={members.members}
          roles={roles ?? []}
          isRemoving={removeMember.isPending}
          onRoleChange={(userId, roleId) => changeRole.mutate({ userId, roleId })}
          onRemove={(userId) => removeMember.mutate(userId)}
        />
      )}
      <InvitationsFeed />
    </SettingsCard>
  );
};
