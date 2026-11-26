import { useNavigate } from "react-router";

import { EmptyRow } from "@/components/EmptyRow";
import { ErrorDisplay } from "@/components/ErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { LogsView } from "@/components/logs/LogsView";
import { LogServicePicker } from "@/components/stack/LogServicePicker";
import { useFetchStackServices } from "@/hooks/StackHooks";
import { useWorkspacePath } from "@/hooks/useWorkspacePath";
import { stackLogSource } from "@/models/ContainerLog";

interface StackLogsTabsProps {
  stackId: string;
  service: string | undefined;
}

// One log per service; the URL carries the chosen one, so a Services row can link straight to its logs.
export const StackLogsTabs = ({ stackId, service }: StackLogsTabsProps) => {
  const { data: services, isPending, error } = useFetchStackServices(stackId);
  const navigate = useNavigate();
  const wsPath = useWorkspacePath();
  const active = services?.find((c) => c.name === service) ?? services?.[0];

  return (
    <div className="space-y-4">
      {isPending && <LoadingDisplay />}
      {error && <ErrorDisplay error={error} title="Couldn't load services." />}
      {services && services.length === 0 && <EmptyRow flush>No services found in this stack yet.</EmptyRow>}
      {services && active && (
        <LogsView
          key={active.name}
          source={stackLogSource(stackId, active.name)}
          leading={
            services.length > 1 && (
              <LogServicePicker services={services} value={active.name} onChange={(name) => navigate(wsPath(`/stacks/${stackId}/logs/${name}`))} />
            )
          }
        />
      )}
    </div>
  );
};
