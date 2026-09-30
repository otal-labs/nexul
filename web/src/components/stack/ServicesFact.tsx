import { useFetchStackServices } from "@/hooks/StackHooks";
import { ContainerStatus } from "@/models/Stack";

interface ServicesFactProps {
  stackId: string;
}

const down: string[] = [ContainerStatus.Exited, ContainerStatus.Stopped];

// A compose stack runs several images, so its facts grid counts services instead of naming one image.
export const ServicesFact = ({ stackId }: ServicesFactProps) => {
  const { data: services } = useFetchStackServices(stackId);
  const stopped = (services ?? []).filter((c) => down.includes(c.status)).length;

  return (
    <>
      {!services && <span className="text-muted-foreground">—</span>}
      {services && (
        <span className="font-mono text-xs">{services.length === 1 ? "1 service" : `${services.length} services`}</span>
      )}
      {stopped > 0 && <span className="text-xs text-destructive">{stopped} not running</span>}
    </>
  );
};
