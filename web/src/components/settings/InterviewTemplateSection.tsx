import { ErrorDisplay } from "@/components/ErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { INTERVIEW_TEMPLATE_CARD, InterviewTemplateForm } from "@/components/settings/InterviewTemplateForm";
import { SettingsCard } from "@/components/settings/SettingsCard";
import { useFetchInterviewTemplate } from "@/hooks/MemoryHooks";
import { useHasPermission } from "@/hooks/WorkspaceHooks";
import { useWorkspaceStore } from "@/stores/workspaceStore";

// The loaded form draws its own card, so its Save can sit in the card's footer beside the text it saves.
export const InterviewTemplateSection = () => {
  const workspaceId = useWorkspaceStore((s) => s.selectedWorkspaceId);
  const canWrite = useHasPermission("memories:write");
  const { data: template, error, isPending } = useFetchInterviewTemplate(workspaceId);

  return (
    <>
      {isPending && (
        <SettingsCard {...INTERVIEW_TEMPLATE_CARD}>
          <LoadingDisplay />
        </SettingsCard>
      )}
      {error && (
        <SettingsCard {...INTERVIEW_TEMPLATE_CARD}>
          <ErrorDisplay error={error} />
        </SettingsCard>
      )}
      {template && <InterviewTemplateForm key={`${template.edited}:${template.body}`} template={template} canWrite={canWrite} />}
    </>
  );
};
