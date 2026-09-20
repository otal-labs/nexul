import {
  Bug,
  FileText,
  FlaskConical,
  ListTodo,
  Milestone,
  Palette,
  Shapes,
  Sparkles,
  Wrench,
  type LucideIcon,
} from "lucide-react";

// TicketType has no icon field, so the icon is inferred from the type name with a generic fallback.
const TICKET_TYPE_ICONS: Record<string, LucideIcon> = {
  task: ListTodo,
  bug: Bug,
  feature: Sparkles,
  chore: Wrench,
  docs: FileText,
  documentation: FileText,
  epic: Milestone,
  spike: FlaskConical,
  design: Palette,
};

export const ticketTypeIcon = (typeName: string): LucideIcon =>
  TICKET_TYPE_ICONS[typeName.trim().toLowerCase()] ?? Shapes;
