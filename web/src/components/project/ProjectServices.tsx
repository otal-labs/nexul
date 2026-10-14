import { EmptyRow } from "@/components/EmptyRow";
import { ErrorDisplay } from "@/components/ErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { WaitingRepositoryRow } from "@/components/project/WaitingRepositoryRow";
import { ServicesFeed } from "@/components/service/ServicesFeed";
import { SettingsCard } from "@/components/settings/SettingsCard";
import { AddServiceLink } from "@/components/wizard/AddServiceLink";
import { useAreaAccess } from "@/hooks/AccessHooks";
import { useFetchServices } from "@/hooks/ServiceHooks";
import { useWaitingRepos } from "@/hooks/useWaitingRepos";
import { canRunWizard } from "@/models/Access";

interface ProjectServicesProps {
  projectId: string;
}

export const ProjectServices = ({ projectId }: ProjectServicesProps) => {
  const { data: services, isPending, error } = useFetchServices(projectId);
  const can = useAreaAccess(projectId);
  const canAdd = !!can && canRunWizard(can, false);
  const waiting = useWaitingRepos(projectId);

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
      {services && services.length === 0 && !waiting?.length && (
        <EmptyRow>No services yet. Create one to set up its first deploy.</EmptyRow>
      )}
      {services && services.length > 0 && <ServicesFeed services={services} />}
      {services && waiting && waiting.length > 0 && (
        <ul className="divide-y divide-border overflow-hidden rounded-md border border-border not-first:mt-4">
          {waiting.map((repo) => (
            <WaitingRepositoryRow key={repo.full_name} projectId={projectId} repo={repo} />
          ))}
        </ul>
      )}
    </SettingsCard>
  );
};
