import { Microheader } from "@/components/Microheader";
import { InstanceTemplateRow } from "@/components/templates/InstanceTemplateRow";
import type { Template } from "@/models/Template";

interface InstanceTemplateGroupProps {
  label: string;
  hint: string;
  templates: Template[];
}

export const InstanceTemplateGroup = ({ label, hint, templates }: InstanceTemplateGroupProps) => (
  <section aria-label={label}>
    <div className="flex flex-wrap items-baseline justify-between gap-x-3 gap-y-0.5 border-b border-border pb-1.5">
      <Microheader>{label}</Microheader>
      <p className="text-xs text-muted-foreground">{hint}</p>
    </div>
    <ul className="divide-y divide-border">
      {templates.map((template) => (
        <InstanceTemplateRow key={`${template.kind}/${template.key}`} template={template} />
      ))}
    </ul>
  </section>
);
