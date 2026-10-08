import { InstanceTemplateRow } from "@/components/templates/InstanceTemplateRow";
import type { Template } from "@/models/Template";

interface InstanceTemplateGroupProps {
  label: string;
  templates: Template[];
}

export const InstanceTemplateGroup = ({ label, templates }: InstanceTemplateGroupProps) => (
  <section aria-label={label}>
    <h3 className="border-b border-border pb-1.5 font-mono text-[11px] uppercase tracking-wide text-muted-foreground">{label}</h3>
    <ul className="divide-y divide-border">
      {templates.map((template) => (
        <InstanceTemplateRow key={`${template.kind}/${template.key}`} template={template} />
      ))}
    </ul>
  </section>
);
