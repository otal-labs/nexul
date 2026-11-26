import { ErrorDisplay } from "@/components/ErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { SettingsCard } from "@/components/settings/SettingsCard";
import { InstanceTemplateGroup } from "@/components/templates/InstanceTemplateGroup";
import { useFetchTemplates } from "@/hooks/TemplateHooks";
import { TEMPLATE_GROUPS } from "@/models/Template";

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
          {TEMPLATE_GROUPS.map((group) => (
            <InstanceTemplateGroup
              key={group.label}
              group={group}
              templates={templates.filter((t) => (group.kinds as string[]).includes(t.kind))}
            />
          ))}
        </div>
      )}
    </SettingsCard>
  );
};
