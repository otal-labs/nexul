import { UnlinkButton } from "@/components/ticket/UnlinkButton";
import { useUnlinkPR } from "@/hooks/TicketHooks";
import type { PRLink } from "@/models/Ticket";

interface UnlinkPRButtonProps {
  ticketId: string;
  pr: PRLink;
}

export const UnlinkPRButton = ({ ticketId, pr }: UnlinkPRButtonProps) => {
  const unlink = useUnlinkPR();
  return <UnlinkButton subject={`pull request #${pr.number}`} onUnlink={() => unlink.mutate({ id: ticketId, pr })} />;
};
