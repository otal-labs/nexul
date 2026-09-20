import { ServicesFeed } from "@/components/service/ServicesFeed";
import { AddServiceLink } from "@/components/wizard/AddServiceLink";
import { useFetchServices } from "@/hooks/ServiceHooks";

interface ProjectServicesProps {
  projectId: string;
}

export const ProjectServices = ({ projectId }: ProjectServicesProps) => {
  const { data: services } = useFetchServices(projectId);

  return (
    <div className="mt-3">
      <div className="flex items-center justify-between gap-2">
        <span className="text-sm font-medium">Services</span>
        <AddServiceLink projectId={projectId} variant="ghost" size="sm">
          New service
        </AddServiceLink>
      </div>
      {services && services.length === 0 && (
        <p className="mt-2 text-sm text-muted-foreground">
          No services in this project yet — create one to define its first deploy.
        </p>
      )}
      {services && services.length > 0 && <ServicesFeed services={services} />}
    </div>
  );
};
