import { CreateInvitationDialog } from "@/components/member/CreateInvitationDialog";
import { InvitationsFeed } from "@/components/member/InvitationsFeed";
import { MembersFeed } from "@/components/member/MembersFeed";
import { NoDataDisplay } from "@/components/NoDataDisplay";
import { PageHeader } from "@/components/PageHeader";
import type { MembersList } from "@/models/Member";
import type { Role } from "@/models/Role";

interface MembersPageContentProps {
  members: MembersList;
  roles: Role[];
  isRemoving: boolean;
  onRoleChange: (userId: string, roleId: string) => void;
  onRemove: (userId: string) => void;
}

export const MembersPageContent = ({
  members,
  roles,
  isRemoving,
  onRoleChange,
  onRemove,
}: MembersPageContentProps) => (
  <>
    <PageHeader
      title="Members"
      subtitle="Manage workspace membership and create private invitation links with access across one or more workspaces."
      actions={<CreateInvitationDialog />}
    />
    <InvitationsFeed />
    {members.members.length === 0 && (
      <NoDataDisplay
        className="mt-4"
        message="No members yet — create an invitation link to add someone."
      />
    )}
    {members.members.length > 0 && (
      <MembersFeed
        members={members.members}
        roles={roles}
        isRemoving={isRemoving}
        onRoleChange={onRoleChange}
        onRemove={onRemove}
      />
    )}
  </>
);
