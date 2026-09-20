import { InviteRow } from "@/components/member/InviteRow";
import { MemberRow } from "@/components/member/MemberRow";
import type { MemberView, WorkspaceInvite } from "@/models/Member";
import type { Role } from "@/models/Role";

interface MembersFeedProps {
  members: MemberView[];
  invites: WorkspaceInvite[];
  roles: Role[];
  isRemoving: boolean;
  isCancelling: boolean;
  onRoleChange: (userId: string, roleId: string) => void;
  onRemove: (userId: string) => void;
  onCancelInvite: (login: string) => void;
}

export const MembersFeed = ({
  members,
  invites,
  roles,
  isRemoving,
  isCancelling,
  onRoleChange,
  onRemove,
  onCancelInvite,
}: MembersFeedProps) => {
  const roleName = (roleId: string) => roles.find((r) => r.id === roleId)?.name ?? roleId;
  // The Owner role is never assignable via this screen, so it never appears as a select option.
  const assignableRoles = roles.filter((r) => !r.is_owner_role);

  return (
    <ul className="mt-4 divide-y divide-border overflow-hidden rounded-md border bg-card shadow-card">
      {members.map((member, index) => (
        <MemberRow
          key={member.user_id}
          userId={member.user_id}
          login={member.login}
          roleId={member.role_id}
          roles={assignableRoles}
          isOwner={roles.find((r) => r.id === member.role_id)?.is_owner_role ?? false}
          isRemoving={isRemoving}
          onRoleChange={onRoleChange}
          onRemove={onRemove}
          index={index}
        />
      ))}
      {invites.map((invite, index) => (
        <InviteRow
          key={invite.login}
          login={invite.login}
          roleName={roleName(invite.role_id)}
          isCancelling={isCancelling}
          onCancel={onCancelInvite}
          index={members.length + index}
        />
      ))}
    </ul>
  );
};
