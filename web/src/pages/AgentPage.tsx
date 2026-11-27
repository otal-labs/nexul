import { Navigate, useLocation, useParams } from "react-router";

import { AgentHeader } from "@/components/agent/AgentHeader";
import { AgentTabs } from "@/components/agent/AgentTabs";
import { PlayTabFeed } from "@/components/agent/PlayTabFeed";
import { Container } from "@/components/Container";
import { NoDataDisplay } from "@/components/NoDataDisplay";
import { useAreaAccess } from "@/hooks/AccessHooks";
import { useWorkspacePath } from "@/hooks/useWorkspacePath";
import { agentTabPath, isAgentTab } from "@/models/Agent";

// The Agent's plays, one tab per type, each play a row that unfolds its settings and its auto plays.
export const AgentPage = () => {
  const { tab } = useParams();
  const can = useAreaAccess();
  const wsPath = useWorkspacePath();
  const { search } = useLocation();

  return (
    <Container className="py-8 sm:pr-3.5">
      {!isAgentTab(tab) && <Navigate replace to={{ pathname: wsPath(agentTabPath("ticket")), search }} />}
      {isAgentTab(tab) && (
        <div className="space-y-2">
          <AgentHeader tab={tab} />
          <AgentTabs active={tab} />
          {can && !can("plays") && <NoDataDisplay message="You can't read plays here" />}
          {can?.("plays") && <PlayTabFeed tab={tab} />}
        </div>
      )}
    </Container>
  );
};
