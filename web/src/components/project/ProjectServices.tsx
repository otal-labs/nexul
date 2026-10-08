import { EmptyRow } from "@/components/EmptyRow";
import { ServicesFeed } from "@/components/service/ServicesFeed";
import { SettingsCard } from "@/components/settings/SettingsCard";
import { AddServiceLink } from "@/components/wizard/AddServiceLink";
import { useAreaAccess } from "@/hooks/AccessHooks";
import { useFetchServices } from "@/hooks/ServiceHooks";

interface ProjectServicesProps {
  projectId: string;
}

export const ProjectServices = ({ projectId }: ProjectServicesProps) => {
  const { data: services } = useFetchServices(projectId);
  const canAdd = useAreaAccess()?.("newProject") ?? false;

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
      {services && services.length === 0 && (
        <EmptyRow>No services yet. Create one to set up its first deploy.</EmptyRow>
      )}
      {services && services.length > 0 && <ServicesFeed services={services} />}
    </SettingsCard>
  );
};
