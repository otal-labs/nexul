import { UnlinkButton } from "@/components/ticket/UnlinkButton";
import { useUnlinkBranch } from "@/hooks/TicketHooks";
import type { BranchLink } from "@/models/Ticket";

interface UnlinkBranchButtonProps {
  ticketId: string;
  branch: BranchLink;
}

export const UnlinkBranchButton = ({ ticketId, branch }: UnlinkBranchButtonProps) => {
  const unlink = useUnlinkBranch();
  return <UnlinkButton subject={`branch ${branch.branch}`} onUnlink={() => unlink.mutate({ id: ticketId, branch })} />;
};
