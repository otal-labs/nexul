import { ErrorDisplay } from "@/components/ErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { ProjectLinkForm } from "@/components/settings/ProjectLinkForm";
import { SettingsCard } from "@/components/settings/SettingsCard";
import { useFetchProjectLink, useListComputers } from "@/hooks/PairingHooks";

interface ProjectLinkSectionProps {
  projectId: string;
}

export const ProjectLinkSection = ({ projectId }: ProjectLinkSectionProps) => {
  const { data: computers, isPending: computersPending, error: computersError } = useListComputers();
  const { data: link, isPending: linkPending, error: linkError } = useFetchProjectLink(projectId);
  const isPending = computersPending || linkPending;
  const error = computersError || linkError;

  return (
    <SettingsCard
      id="pairing"
      title="T3 pairing"
      description="Which paired computer, T3 project, and provider/model @Agent runs against inside this
        project — leave unlinked to fall back to each teammate's own pairing defaults."
    >
      {isPending && <LoadingDisplay />}
      {error && <ErrorDisplay error={error} />}
      {computers && link && <ProjectLinkForm projectId={projectId} link={link} computers={computers} />}
    </SettingsCard>
  );
};
