import { EmptyRow } from "@/components/EmptyRow";
import { ErrorDisplay } from "@/components/ErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { ServicesFeed } from "@/components/service/ServicesFeed";
import { SettingsCard } from "@/components/settings/SettingsCard";
import { AddServiceLink } from "@/components/wizard/AddServiceLink";
import { useAreaAccess } from "@/hooks/AccessHooks";
import { useFetchServices } from "@/hooks/ServiceHooks";
import { ServicesWaitingRows, useDeployPromptVariant, useWaitingRepos } from "@/prototype/projectpaths/DeployPrompts";

interface ProjectServicesProps {
  projectId: string;
}

export const ProjectServices = ({ projectId }: ProjectServicesProps) => {
  const { data: services, isPending, error } = useFetchServices(projectId);
  const canAdd = useAreaAccess()?.("newProject") ?? false;
  const waiting = useWaitingRepos(projectId);
  const showWaiting = useDeployPromptVariant() === "a" && waiting.length > 0;

  return (
    <SettingsCard
      id="services"
      title="Services"
      description="What this project deploys, each with its own deploys, logs, and hostnames."
      footer={
        canAdd && (
          <AddServiceLink projectId={projectId} variant="outline" size="sm">
            New service
          </AddServiceLink>
        )
      }
    >
      {isPending && <LoadingDisplay />}
      {error && <ErrorDisplay error={error} />}
      {showWaiting && <ServicesWaitingRows projectId={projectId} repos={waiting} />}
      {services && services.length === 0 && !showWaiting && (
        <EmptyRow>No services yet. Create one to set up its first deploy.</EmptyRow>
      )}
      {services && services.length > 0 && <ServicesFeed services={services} />}
    </SettingsCard>
  );
};
