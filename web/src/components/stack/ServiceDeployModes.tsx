import type { Container } from "@/models/Stack";
import { deployModes } from "@/utils/DeploySwapUtility";

interface ServiceDeployModesProps {
  services: Container[];
  hostnamesOf: (id: string) => string[];
}

// Which services the next deploy swaps with no downtime, and which it restarts in place and why.
export const ServiceDeployModes = ({ services, hostnamesOf }: ServiceDeployModesProps) => {
  const { swapped, restarted } = deployModes(services, hostnamesOf);
  const restartText = restarted.map((g) => `${g.names.join(", ")} (${g.reason})`).join(" · ");
  return (
    <div className="mt-1 space-y-0.5 text-xs text-muted-foreground">
      {swapped.length > 0 && <p>{`Deploys swap with no downtime: ${swapped.join(", ")}`}</p>}
      {restarted.length > 0 && <p>{`Deploys restart in place: ${restartText}`}</p>}
    </div>
  );
};
