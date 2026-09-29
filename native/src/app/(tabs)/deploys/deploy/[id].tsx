import { AreaGate } from "@/components/AreaGate";
import { DeployScreen } from "@/components/deploys/DeployScreen";

export default function DeployRoute() {
  return (
    <AreaGate area="deploys">
      <DeployScreen />
    </AreaGate>
  );
}
