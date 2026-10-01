import { FileTextIcon, Link2Icon, MessageSquareIcon, TicketIcon, type LucideIcon } from "lucide-react";
import { Link } from "react-router";

import { useInstanceLinkLabel } from "@/hooks/useInstanceLinkLabel";
import { parseInstanceLink, type InstanceLink } from "@/utils/MessageTextUtility";

const SECTION_ICONS: Record<string, LucideIcon> = { chat: MessageSquareIcon, docs: FileTextIcon, tickets: TicketIcon };

interface InstanceLinkPillProps {
  url: string;
  link: InstanceLink;
}

// data-url lets a copied selection carry the URL instead of the pill's label (MessageBody's onCopy).
const InstanceLinkPill = ({ url, link }: InstanceLinkPillProps) => {
  const label = useInstanceLinkLabel(link);
  const Icon = SECTION_ICONS[link.section ?? ""] ?? Link2Icon;
  return (
    <Link to={link.path} title={url} data-url={url} className="mention-chip">
      <Icon className="size-[1.1em] shrink-0" aria-hidden />
      <span className="mention-chip__label">{label}</span>
    </Link>
  );
};

const ExternalLink = ({ url }: { url: string }) => (
  <a href={url} target="_blank" rel="noopener noreferrer" className="break-all underline underline-offset-2">
    {url}
  </a>
);

// A link into this instance navigates in-app as a pill; anything else opens in a new tab.
export const MessageLink = ({ url }: { url: string }) => {
  const link = parseInstanceLink(url, window.location.origin);
  return (
    <>
      {link && <InstanceLinkPill url={url} link={link} />}
      {!link && <ExternalLink url={url} />}
    </>
  );
};
