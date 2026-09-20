import { InviteQueueSection } from "@/components/member/InviteQueueSection";
import { MembersFeed } from "@/components/member/MembersFeed";
import { NoAssignableRolesEmptyState } from "@/components/member/NoAssignableRolesEmptyState";
import { NoDataDisplay } from "@/components/NoDataDisplay";
import { PageHeader } from "@/components/PageHeader";
import type { MembersList, PendingInvite } from "@/models/Member";
import type { Role } from "@/models/Role";

interface MembersPageContentProps {
  members: MembersList;
  roles: Role[];
  assignableRoles: Role[];
  isRemoving: boolean;
  isCancelling: boolean;
  onInvite: (entry: PendingInvite) => Promise<unknown>;
  onRoleChange: (userId: string, roleId: string) => void;
  onRemove: (userId: string) => void;
  onCancelInvite: (login: string) => void;
}

export const MembersPageContent = ({
  members,
  roles,
  assignableRoles,
  isRemoving,
  isCancelling,
  onInvite,
  onRoleChange,
  onRemove,
  onCancelInvite,
}: MembersPageContentProps) => (
  <>
    <PageHeader
      title="Members"
      subtitle="Invite people into this workspace and assign the role they hold here. Someone can hold a different role in each workspace they belong to."
    />
    {assignableRoles.length === 0 && <NoAssignableRolesEmptyState />}
    {assignableRoles.length > 0 && (
      <InviteQueueSection
        assignableRoles={assignableRoles}
        existingLogins={[...members.invites.map((i) => i.login), ...members.members.map((m) => m.login)]}
        onInvite={onInvite}
      />
    )}
    {members.members.length === 0 && members.invites.length === 0 && (
      <NoDataDisplay
        className="mt-4"
        message="No members yet — invite an allowlisted GitHub username to add them to this workspace."
      />
    )}
    {(members.members.length > 0 || members.invites.length > 0) && (
      <MembersFeed
        members={members.members}
        invites={members.invites}
        roles={roles}
        isRemoving={isRemoving}
        isCancelling={isCancelling}
        onRoleChange={onRoleChange}
        onRemove={onRemove}
        onCancelInvite={onCancelInvite}
      />
    )}
  </>
);
