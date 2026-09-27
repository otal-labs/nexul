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
  type LucideProps,
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

export const TicketTypeIcon = ({ typeName, ...props }: LucideProps & { typeName: string }) => {
  const Icon = TICKET_TYPE_ICONS[typeName.trim().toLowerCase()] ?? Shapes;
  return <Icon {...props} />;
};
