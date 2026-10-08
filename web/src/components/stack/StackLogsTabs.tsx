import { useNavigate } from "react-router";

import { EmptyRow } from "@/components/EmptyRow";
import { ErrorDisplay } from "@/components/ErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { LogsView } from "@/components/logs/LogsView";
import { Tabs, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { TabUnderline } from "@/components/ActiveIndicator";
import { useFetchStackServices } from "@/hooks/StackHooks";
import { useWorkspacePath } from "@/hooks/useWorkspacePath";

interface StackLogsTabsProps {
  stackId: string;
  service: string | undefined;
}

// One tab per service; the URL carries the chosen one, so a Services row can link straight to its logs.
export const StackLogsTabs = ({ stackId, service }: StackLogsTabsProps) => {
  const { data: services, isPending, error } = useFetchStackServices(stackId);
  const navigate = useNavigate();
  const wsPath = useWorkspacePath();
  const active = services?.find((c) => c.name === service) ?? services?.[0];

  return (
    <div className="space-y-4">
      {isPending && <LoadingDisplay />}
      {error && <ErrorDisplay error={error} title="Could not load services" />}
      {services && services.length === 0 && <EmptyRow flush>No services parsed for this stack yet.</EmptyRow>}
      {services && active && (
        <Tabs value={active.name} onValueChange={(name) => navigate(wsPath(`/stacks/${stackId}/logs/${name}`))}>
          <TabsList variant="line" aria-label="Service" className="relative isolate w-full justify-start overflow-x-auto border-b border-border p-0">
            <TabUnderline />
            {services.map((c) => (
              <TabsTrigger
                key={c.id}
                value={c.name}
                className="flex-none px-3 font-mono after:hidden"
              >
                {c.name}
              </TabsTrigger>
            ))}
          </TabsList>
        </Tabs>
      )}
      {active && <LogsView key={active.name} stackId={stackId} service={active.name} />}
    </div>
  );
};
