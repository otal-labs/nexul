import { FileTextIcon, LockIcon, TicketIcon, type LucideIcon } from "lucide-react";
import { Link } from "react-router";

import { useSelectedWorkspace } from "@/hooks/WorkspaceHooks";
import { cn } from "@/lib/utils";
import type { MentionChipData, MentionType } from "@/models/Mention";
import { DEFAULT_MENTION_CHIP_TEMPLATE } from "@/models/Workspace";

interface MentionChipProps {
  type: MentionType;
  id: string;
  label: string;
  chip?: MentionChipData | undefined;
}

// Substitutes {ticket.Field} tokens (spec.md §6); unrecognized tokens pass through unchanged.
const renderTicketTemplate = (template: string, chip: MentionChipData | undefined, title: string) => {
  const project = chip?.project_prefix && chip.project_number !== undefined ? `${chip.project_prefix}-${chip.project_number}` : "";
  const values: Record<string, string> = {
    Project: project,
    Ticket: title,
    Status: chip?.status_label ?? "",
    Type: chip?.type_label ?? "",
    Developer: chip?.developer_label ?? "",
    Due: chip?.due_label ?? "",
  };
  return template.replace(/\{ticket\.(\w+)\}/g, (match, key: string) => values[key] ?? match);
};

const MENTION_ICONS: Record<MentionType, LucideIcon> = { ticket: TicketIcon, doc: FileTextIcon };

const hrefFor = (type: MentionType, id: string) => (type === "ticket" ? `/tickets/${id}` : `/docs/${id}`);

const textFor = (type: MentionType, template: string, chip: MentionChipData | undefined, title: string) =>
  type === "ticket" ? renderTicketTemplate(template, chip, title) : title;

// Ticket text follows the workspace template (spec.md §6); inert with disclosed title if not openable.
export const MentionChip = ({ type, id, label, chip }: MentionChipProps) => {
  const workspace = useSelectedWorkspace();
  const title = chip?.title || label;
  const canOpen = chip?.can_open ?? false;
  const Icon = MENTION_ICONS[type];
  const template = workspace?.mention_chip_template || DEFAULT_MENTION_CHIP_TEMPLATE;
  const text = textFor(type, template, chip, title);

  const content = (
    <>
      <Icon className="h-3.5 w-3.5 shrink-0" />
      <span className="min-w-0 truncate">{text}</span>
      {!canOpen && <LockIcon className="h-3 w-3 shrink-0 text-muted-foreground" />}
    </>
  );

  return (
    <>
      {canOpen && (
        <Link
          to={hrefFor(type, id)}
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
        <span
          className={cn("mention-chip", "mention-chip--inert")}
          data-testid="mention-chip"
          data-can-open="false"
          data-mention-type={type}
          data-mention-id={id}
        >
          {content}
        </span>
      )}
    </>
  );
};
