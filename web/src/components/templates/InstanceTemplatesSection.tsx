import { ErrorDisplay } from "@/components/ErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { SettingsCard } from "@/components/settings/SettingsCard";
import { InstanceTemplateGroup } from "@/components/templates/InstanceTemplateGroup";
import { useFetchTemplates } from "@/hooks/TemplateHooks";
import { TEMPLATE_GROUP_HINTS, TEMPLATE_KINDS } from "@/models/Template";

export const InstanceTemplatesSection = () => {
  const { data: templates, error, isPending } = useFetchTemplates();

  return (
    <SettingsCard
      id="instance-templates"
      title="Starting templates"
      description="What workspaces and projects start from."
    >
      {isPending && <LoadingDisplay />}
      {error && <ErrorDisplay error={error} />}
      {templates && (
        <div className="space-y-6">
          {TEMPLATE_KINDS.map((kind, index) => (
            <InstanceTemplateGroup
              key={kind}
              kind={kind}
              templates={templates.filter((t) => t.kind === kind)}
              showHint={index === 0 || TEMPLATE_GROUP_HINTS[TEMPLATE_KINDS[index - 1]!] !== TEMPLATE_GROUP_HINTS[kind]}
            />
          ))}
        </div>
      )}
    </SettingsCard>
  );
};
