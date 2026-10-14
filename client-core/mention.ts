// What a doc or ticket mention resolves to through /api/mentions/resolve; people resolve from the workspace's People instead.
export type MentionType = "ticket" | "doc";

export interface MentionRef {
  type: MentionType;
  id: string;
}

export interface MentionChipData {
  type: MentionType;
  id: string;
  title: string;
  status?: string;
  status_label?: string;
  can_open: boolean;
  // Back the mention-chip layout template's tokens; ticket-only, doc chips leave them undefined.
  project_prefix?: string;
  project_number?: number;
  type_label?: string;
  developer_label?: string;
  due_label?: string;
}

export const DEFAULT_MENTION_CHIP_TEMPLATE = "{ticket.Ticket} {ticket.Status}";

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

// A ticket follows the workspace's {ticket.Field} template and a doc shows its title; unresolved, the stored label stands in.
export const mentionChipText = (type: MentionType, template: string, chip: MentionChipData | undefined, label: string): string => {
  const title = chip?.title || label;
  if (type === "ticket") return renderTicketTemplate(template || DEFAULT_MENTION_CHIP_TEMPLATE, chip, title);
  return title;
};
