import { ComputersButton } from "@/components/agent/ComputersButton";
import { NewPlayButton } from "@/components/agent/NewPlayButton";
import { PageHeader } from "@/components/PageHeader";
import type { AgentTab } from "@/models/Agent";

export const AgentHeader = ({ tab }: { tab: AgentTab }) => (
  <PageHeader
    title="Agent"
    className="border-b-0 pb-4"
    actions={
      <>
        <ComputersButton />
        <NewPlayButton tab={tab} />
      </>
    }
  />
);
