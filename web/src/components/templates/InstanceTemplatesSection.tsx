import { ErrorDisplay } from "@/components/ErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { SettingsCard } from "@/components/settings/SettingsCard";
import { InstanceTemplateGroup } from "@/components/templates/InstanceTemplateGroup";
import { useFetchTemplates } from "@/hooks/TemplateHooks";
import { TEMPLATE_GROUP_LABELS, TEMPLATE_KINDS } from "@/models/Template";

export const InstanceTemplatesSection = () => {
  const { data: templates, error, isPending } = useFetchTemplates();

  return (
    <SettingsCard
      id="instance-templates"
      title="Templates"
      description="What every workspace and project starts from. Interview and mention chip templates follow these until a workspace edits its own. Play instructions and ticket bodies are copied in when a workspace or project is created. The intro and footer open and close every Agent prompt."
    >
      {isPending && <LoadingDisplay />}
      {error && <ErrorDisplay error={error} />}
      {templates && (
        <div className="space-y-6">
          {TEMPLATE_KINDS.map((kind) => (
            <InstanceTemplateGroup
              key={kind}
              label={TEMPLATE_GROUP_LABELS[kind]}
              templates={templates.filter((t) => t.kind === kind)}
            />
          ))}
        </div>
      )}
    </SettingsCard>
  );
};
