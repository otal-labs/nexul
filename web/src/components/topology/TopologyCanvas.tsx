import { ReactFlowProvider } from "@xyflow/react";

import { TopologyFlow } from "@/components/topology/TopologyFlow";

// Provider boundary lives here, React Flow wiring lives in TopologyFlow, so each file owns one concern.
export const TopologyCanvas = () => (
  <ReactFlowProvider>
    <TopologyFlow />
  </ReactFlowProvider>
);
