import { Container } from "@/components/Container";
import { ErrorDisplay } from "@/components/ErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { MembersPageContent } from "@/components/member/MembersPageContent";
import {
  useCancelWorkspaceInvite,
  useChangeWorkspaceMemberRole,
  useFetchWorkspaceMembers,
  useInviteWorkspaceMember,
  useRemoveWorkspaceMember,
} from "@/hooks/MemberHooks";
import { useFetchWorkspaceRoles } from "@/hooks/RoleHooks";
import { useWorkspaceStore } from "@/stores/workspaceStore";

export const MembersPage = () => {
  const selectedWorkspaceId = useWorkspaceStore((s) => s.selectedWorkspaceId);
  const { data: members, isPending, error } = useFetchWorkspaceMembers(selectedWorkspaceId);
  const { data: roles } = useFetchWorkspaceRoles(selectedWorkspaceId);
  const inviteMember = useInviteWorkspaceMember(selectedWorkspaceId);
  const removeMember = useRemoveWorkspaceMember(selectedWorkspaceId);
  const changeRole = useChangeWorkspaceMemberRole(selectedWorkspaceId);
  const cancelInvite = useCancelWorkspaceInvite(selectedWorkspaceId);

  // Excludes the singleton Owner role since it isn't invite-assignable, so the picker never offers it by mistake.
  const assignableRoles = (roles ?? []).filter((r) => !r.is_owner_role);

  return (
    <Container className="mx-auto max-w-3xl py-8">
      {isPending && <LoadingDisplay />}
      {error && <ErrorDisplay error={error} />}
      {members && (
        <MembersPageContent
          members={members}
          roles={roles ?? []}
          assignableRoles={assignableRoles}
          isRemoving={removeMember.isPending}
          isCancelling={cancelInvite.isPending}
          onInvite={(entry) => inviteMember.mutateAsync({ login: entry.login, roleId: entry.roleId })}
          onRoleChange={(userId, roleId) => changeRole.mutate({ userId, roleId })}
          onRemove={(userId) => removeMember.mutate(userId)}
          onCancelInvite={(login) => cancelInvite.mutate(login)}
        />
      )}
    </Container>
  );
};
