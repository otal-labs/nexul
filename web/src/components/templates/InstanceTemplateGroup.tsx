import { Microheader } from "@/components/Microheader";
import { SettingsHint } from "@/components/settings/SettingsHint";
import { settingsListClass } from "@/components/settings/SettingsRow";
import { InstanceTemplateRow } from "@/components/templates/InstanceTemplateRow";
import { NewMemoryTemplate } from "@/components/templates/NewMemoryTemplate";
import { useRowGlide } from "@/hooks/useRowGlide";
import { MEMORY_TEMPLATE, TEMPLATE_GROUP_HINTS, TEMPLATE_GROUP_LABELS, type Template, type TextTemplateKind } from "@/models/Template";

interface InstanceTemplateGroupProps {
  kind: TextTemplateKind;
  templates: Template[];
  // False when the group above already said the same thing.
  showHint?: boolean;
}

export const InstanceTemplateGroup = ({ kind, templates, showHint = true }: InstanceTemplateGroupProps) => {
  const { ref: glideRef, prepare: prepareGlide } = useRowGlide();
  const label = TEMPLATE_GROUP_LABELS[kind];

  return (
    <section aria-label={label}>
      <div className="space-y-1 pb-2">
        <Microheader>{label}</Microheader>
        {showHint && <SettingsHint>{TEMPLATE_GROUP_HINTS[kind]}</SettingsHint>}
      </div>
      <ul ref={glideRef} className={settingsListClass}>
        {templates.map((template) => (
          <InstanceTemplateRow key={`${template.kind}/${template.key}`} template={template} onLeave={prepareGlide} />
        ))}
      </ul>
      {kind === MEMORY_TEMPLATE && <NewMemoryTemplate />}
    </section>
  );
};
