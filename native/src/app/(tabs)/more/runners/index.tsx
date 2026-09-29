import { AreaGate } from "@/components/AreaGate";
import { RunnersScreen } from "@/components/runners/RunnersScreen";

export default function RunnersRoute() {
  return (
    <AreaGate area="runners">
      <RunnersScreen />
    </AreaGate>
  );
}
