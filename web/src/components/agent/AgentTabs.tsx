import { useNavigate } from "react-router";

import { TabUnderline } from "@/components/ActiveIndicator";
import { Tabs, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { useFetchWorkspacePlays } from "@/hooks/PlayHooks";
import { useWorkspacePath } from "@/hooks/useWorkspacePath";
import { AGENT_TABS, AGENT_TAB_LABELS, agentTabPath, type AgentTab } from "@/models/Agent";
import { useWorkspaceStore } from "@/stores/workspaceStore";

// One tab per play type, each a path so a link lands on it, with how many plays it holds.
export const AgentTabs = ({ active }: { active: AgentTab }) => {
  const navigate = useNavigate();
  const wsPath = useWorkspacePath();
  const workspaceId = useWorkspaceStore((s) => s.selectedWorkspaceId);
  const { data: plays } = useFetchWorkspacePlays(workspaceId);

  return (
    <Tabs value={active} onValueChange={(tab) => void navigate(wsPath(agentTabPath(tab as AgentTab)))}>
      <TabsList variant="line" aria-label="Play type" className="relative isolate w-full justify-start overflow-x-auto overflow-y-hidden scrollbar-none border-b border-border p-0">
        <TabUnderline />
        {AGENT_TABS.map((tab) => (
          <TabsTrigger key={tab} value={tab} className="flex-none gap-1.5 px-3 after:hidden">
            {AGENT_TAB_LABELS[tab]}
            {plays && <span className="font-mono text-xs text-muted-foreground tabular-nums">{plays.filter((play) => play.type === tab).length}</span>}
          </TabsTrigger>
        ))}
      </TabsList>
    </Tabs>
  );
};
