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
  // Split keeps the tokens at the odd indexes; whitespace alone between two fields reads as a dot, and an empty field takes no dot.
  const parts = template.split(/(\{ticket\.\w+\})/);
  let text = "";
  let between = "";
  let afterField = false;
  for (const [index, part] of parts.entries()) {
    if (index % 2 === 0) {
      between += part;
      continue;
    }
    const value = values[part.slice("{ticket.".length, -1)] ?? part;
    if (!value) continue;
    text += afterField && between.trim() === "" ? " · " : between;
    text += value;
    between = "";
    afterField = true;
  }
  return (text + between).trim();
};

// A ticket follows the workspace's {ticket.Field} template and a doc shows its title; unresolved, the stored label stands in.
export const mentionChipText = (type: MentionType, template: string, chip: MentionChipData | undefined, label: string): string => {
  const title = chip?.title || label;
  if (type === "ticket") return renderTicketTemplate(template || DEFAULT_MENTION_CHIP_TEMPLATE, chip, title);
  return title;
};
