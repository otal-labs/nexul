import { Microheader } from "@/components/Microheader";
import { InstanceTemplateRow } from "@/components/templates/InstanceTemplateRow";
import type { Template } from "@/models/Template";

interface InstanceTemplateGroupProps {
  label: string;
  templates: Template[];
}

export const InstanceTemplateGroup = ({ label, templates }: InstanceTemplateGroupProps) => (
  <section aria-label={label}>
    <Microheader className="border-b border-border pb-1.5">{label}</Microheader>
    <ul className="divide-y divide-border">
      {templates.map((template) => (
        <InstanceTemplateRow key={`${template.kind}/${template.key}`} template={template} />
      ))}
    </ul>
  </section>
);
