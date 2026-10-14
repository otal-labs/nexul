import { FileTextIcon, LockIcon, TicketIcon, type LucideIcon } from "lucide-react";
import { Link } from "react-router";

import { mentionChipText, type MentionChipData, type MentionType } from "@nexul/client-core/mention";

import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";

import { useWorkspacePath } from "@/hooks/useWorkspacePath";
import { useSelectedWorkspace } from "@/hooks/WorkspaceHooks";
import { cn } from "@/lib/utils";

interface MentionChipProps {
  type: MentionType;
  id: string;
  label: string;
  chip?: MentionChipData | undefined;
}

const MENTION_ICONS: Record<MentionType, LucideIcon> = { ticket: TicketIcon, doc: FileTextIcon };

const hrefFor = (type: MentionType, id: string) => (type === "ticket" ? `/tickets/${id}` : `/docs/${id}`);

// Ticket text follows the workspace template (spec.md §6); one the viewer can't open shows the same text, locked and inert.
export const MentionChip = ({ type, id, label, chip }: MentionChipProps) => {
  const workspace = useSelectedWorkspace();
  const wsPath = useWorkspacePath();
  const canOpen = chip?.can_open ?? false;
  const Icon = MENTION_ICONS[type];
  const text = mentionChipText(type, workspace?.mention_chip_template ?? "", chip, label);

  const content = (
    <>
      <Icon className="size-[1.1em] shrink-0" />
      <span className="mention-chip__label">{text}</span>
      {!canOpen && <LockIcon role="img" aria-label="You don't have access" className="size-[0.9em] shrink-0 text-muted-foreground" />}
    </>
  );

  return (
    <>
      {canOpen && (
        <Link
          to={wsPath(hrefFor(type, id))}
          className="mention-chip"
          data-testid="mention-chip"
          data-can-open="true"
          data-mention-type={type}
          data-mention-id={id}
        >
          {content}
        </Link>
      )}
      {!canOpen && (
        <Tooltip>
          <TooltipTrigger asChild>
            <span
              className={cn("mention-chip", "mention-chip--inert")}
              data-testid="mention-chip"
              data-can-open="false"
              data-mention-type={type}
              data-mention-id={id}
            >
              {content}
            </span>
          </TooltipTrigger>
          <TooltipContent>You don't have access</TooltipContent>
        </Tooltip>
      )}
    </>
  );
};
