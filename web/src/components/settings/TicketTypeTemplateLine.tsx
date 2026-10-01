import { TemplateOriginLine } from "@/components/templates/TemplateOriginLine";
import { useFetchTemplates } from "@/hooks/TemplateHooks";
import { findTemplate } from "@/models/Template";
import type { TicketType } from "@/models/TicketType";

interface TicketTypeTemplateLineProps {
  type: TicketType;
  projectId: string;
}

// Only a type named like one of the instance's (task, bug, feature) has an instance template to compare with.
export const TicketTypeTemplateLine = ({ type, projectId }: TicketTypeTemplateLineProps) => {
  const { data: templates } = useFetchTemplates();
  const instance = findTemplate(templates, "ticket_body", type.name);
  if (!instance) return null;
  return (
    <TemplateOriginLine
      kind="ticket_body"
      templateKey={instance.key}
      at={{ scope: "project", project_id: projectId }}
      state={type.body_template === instance.body ? "matches" : "differs"}
      canReset
    />
  );
};
