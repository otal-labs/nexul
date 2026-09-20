import { MemberRow } from "@/components/member/MemberRow";
import type { MemberView } from "@/models/Member";
import type { Role } from "@/models/Role";

interface MembersFeedProps {
  members: MemberView[];
  roles: Role[];
  isRemoving: boolean;
  onRoleChange: (userId: string, roleId: string) => void;
  onRemove: (userId: string) => void;
}

export const MembersFeed = ({
  members,
  roles,
  isRemoving,
  onRoleChange,
  onRemove,
}: MembersFeedProps) => {
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
    </ul>
  );
};
