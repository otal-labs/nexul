import { Container } from "@/components/Container";
import { ErrorDisplay } from "@/components/ErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { MembersPageContent } from "@/components/member/MembersPageContent";
import {
  useChangeWorkspaceMemberRole,
  useFetchWorkspaceMembers,
  useRemoveWorkspaceMember,
} from "@/hooks/MemberHooks";
import { useFetchWorkspaceRoles } from "@/hooks/RoleHooks";
import { useWorkspaceStore } from "@/stores/workspaceStore";

export const MembersPage = () => {
  const selectedWorkspaceId = useWorkspaceStore((s) => s.selectedWorkspaceId);
  const { data: members, isPending, error } = useFetchWorkspaceMembers(selectedWorkspaceId);
  const { data: roles } = useFetchWorkspaceRoles(selectedWorkspaceId);
  const removeMember = useRemoveWorkspaceMember(selectedWorkspaceId);
  const changeRole = useChangeWorkspaceMemberRole(selectedWorkspaceId);

  return (
    <Container className="mx-auto max-w-3xl py-8">
      {isPending && <LoadingDisplay />}
      {error && <ErrorDisplay error={error} />}
      {members && (
        <MembersPageContent
          members={members}
          roles={roles ?? []}
          isRemoving={removeMember.isPending}
          onRoleChange={(userId, roleId) => changeRole.mutate({ userId, roleId })}
          onRemove={(userId) => removeMember.mutate(userId)}
        />
      )}
    </Container>
  );
};
