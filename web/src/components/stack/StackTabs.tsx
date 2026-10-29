import { useNavigate } from "react-router";

import { Tabs, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { TabUnderline } from "@/components/ActiveIndicator";
import { useWorkspacePath } from "@/hooks/useWorkspacePath";

const STACK_TABS = ["overview", "deploys", "logs", "settings"] as const;

export type StackTab = (typeof STACK_TABS)[number];

const tabLabels: Record<StackTab, string> = { overview: "Overview", deploys: "Deploys", logs: "Logs", settings: "Settings" };

interface StackTabsProps {
  stackId: string;
  active: StackTab;
  // Container logs carry secrets, so they are read with stacks:logs rather than the stack's own bit.
  showLogs: boolean;
  deployCount: number | undefined;
}

// The daily views first, configuration last; each tab is a path segment so a link can land on it.
export const StackTabs = ({ stackId, active, showLogs, deployCount }: StackTabsProps) => {
  const navigate = useNavigate();
  const wsPath = useWorkspacePath();
  const tabs = STACK_TABS.filter((tab) => tab !== "logs" || showLogs);
  return (
    <Tabs value={active} onValueChange={(tab) => navigate(wsPath(`/stacks/${stackId}${tab === "overview" ? "" : `/${tab}`}`))}>
      <TabsList
        variant="line"
        aria-label="Stack"
        className="relative isolate w-full justify-start overflow-x-auto overflow-y-hidden scrollbar-none border-b border-border p-0"
      >
        <TabUnderline />
        {tabs.map((tab) => (
          <TabsTrigger key={tab} value={tab} className="flex-none gap-1.5 px-3 after:hidden">
            {tabLabels[tab]}
            {tab === "deploys" && !!deployCount && (
              <span className="font-mono text-xs text-muted-foreground tabular-nums">{deployCount}</span>
            )}
          </TabsTrigger>
        ))}
      </TabsList>
    </Tabs>
  );
};
