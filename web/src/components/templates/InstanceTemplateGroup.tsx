import { Microheader } from "@/components/Microheader";
import { SettingsHint } from "@/components/settings/SettingsHint";
import { settingsListClass } from "@/components/settings/SettingsRow";
import { InstanceTemplateRow } from "@/components/templates/InstanceTemplateRow";
import { NewMemoryTemplate } from "@/components/templates/NewMemoryTemplate";
import { useRowGlide } from "@/hooks/useRowGlide";
import { MEMORY_TEMPLATE, type Template, type TEMPLATE_GROUPS } from "@/models/Template";

interface InstanceTemplateGroupProps {
  group: (typeof TEMPLATE_GROUPS)[number];
  templates: Template[];
}

export const InstanceTemplateGroup = ({ group, templates }: InstanceTemplateGroupProps) => {
  const { ref: glideRef, prepare: prepareGlide } = useRowGlide();

  return (
    <section aria-label={group.label}>
      <div className="space-y-1 pb-2">
        <Microheader>{group.label}</Microheader>
        <SettingsHint>{group.hint}</SettingsHint>
      </div>
      <ul ref={glideRef} className={settingsListClass}>
        {templates.map((template) => (
          <InstanceTemplateRow key={`${template.kind}/${template.key}`} template={template} onLeave={prepareGlide} />
        ))}
      </ul>
      {group.kinds.includes(MEMORY_TEMPLATE) && <NewMemoryTemplate />}
    </section>
  );
};
