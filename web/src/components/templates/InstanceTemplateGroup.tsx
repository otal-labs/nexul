import { Microheader } from "@/components/Microheader";
import { InstanceTemplateRow } from "@/components/templates/InstanceTemplateRow";
import { NewMemoryTemplate } from "@/components/templates/NewMemoryTemplate";
import { useRowGlide } from "@/hooks/useRowGlide";
import { MEMORY_TEMPLATE, TEMPLATE_GROUP_HINTS, TEMPLATE_GROUP_LABELS, type Template, type TextTemplateKind } from "@/models/Template";

interface InstanceTemplateGroupProps {
  kind: TextTemplateKind;
  templates: Template[];
}

export const InstanceTemplateGroup = ({ kind, templates }: InstanceTemplateGroupProps) => {
  const { ref: glideRef, prepare: prepareGlide } = useRowGlide();
  const label = TEMPLATE_GROUP_LABELS[kind];

  return (
    <section aria-label={label}>
      <div className="flex flex-wrap items-baseline justify-between gap-x-3 gap-y-0.5 border-b border-border pb-1.5">
        <Microheader>{label}</Microheader>
        <p className="text-xs text-muted-foreground">{TEMPLATE_GROUP_HINTS[kind]}</p>
      </div>
      <ul ref={glideRef} className="divide-y divide-border">
        {templates.map((template) => (
          <InstanceTemplateRow key={`${template.kind}/${template.key}`} template={template} onLeave={prepareGlide} />
        ))}
      </ul>
      {kind === MEMORY_TEMPLATE && <NewMemoryTemplate />}
    </section>
  );
};
