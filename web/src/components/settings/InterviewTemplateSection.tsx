import { ErrorDisplay } from "@/components/ErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { InterviewTemplateForm } from "@/components/settings/InterviewTemplateForm";
import { SettingsCard } from "@/components/settings/SettingsCard";
import { useFetchInterviewTemplate } from "@/hooks/MemoryHooks";
import { useHasPermission } from "@/hooks/WorkspaceHooks";
import { useWorkspaceStore } from "@/stores/workspaceStore";

export const InterviewTemplateSection = () => {
  const workspaceId = useWorkspaceStore((s) => s.selectedWorkspaceId);
  const canWrite = useHasPermission("memories:write");
  const { data: template, error, isPending } = useFetchInterviewTemplate(workspaceId);

  return (
    <SettingsCard
      id="interview-template"
      title="Interview template"
      description="Where every new project's interview starts. A project copies it once: editing the template later never changes an existing interview, and editing an interview never changes the template."
    >
      {isPending && <LoadingDisplay />}
      {error && <ErrorDisplay error={error} />}
      {template && <InterviewTemplateForm key={template.updated_at} template={template} canWrite={canWrite} />}
    </SettingsCard>
  );
};
