import type { CSSProperties } from "react";
import { useNavigate } from "react-router";

import { EmptyRow } from "@/components/EmptyRow";
import { ErrorDisplay } from "@/components/ErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { LogsView } from "@/components/logs/LogsView";
import { Tabs, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { TabUnderline } from "@/components/ActiveIndicator";
import { useFetchStackServices } from "@/hooks/StackHooks";
import { useWorkspacePath } from "@/hooks/useWorkspacePath";
import { stackLogSource } from "@/models/ContainerLog";
import { ContainerStatus } from "@/models/Stack";
import { cn } from "@/lib/utils";

// Each tab says how its service is doing, so a down one is found without opening every tab.
const dotClass: Record<string, string> = {
  [ContainerStatus.Healthy]: "bg-success",
  [ContainerStatus.Running]: "bg-info",
  [ContainerStatus.Pending]: "bg-warning",
  [ContainerStatus.Exited]: "bg-destructive",
};

// The row fades at its right edge so a cut label reads as more; its bottom 2px (the hairline and the active underline)
// stay whole to the edge.
const edgeFade: CSSProperties = {
  maskImage: "linear-gradient(to right, #000 calc(100% - 3rem), transparent), linear-gradient(#000, #000)",
  maskSize: "100% calc(100% - 2px), 100% 2px",
  maskPosition: "top, bottom",
  maskRepeat: "no-repeat",
};

// A deep link can land on a service past the row's right edge; bring its tab into view.
const revealActive = (el: HTMLButtonElement | null) => el?.scrollIntoView?.({ block: "nearest", inline: "nearest" });

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
      {error && <ErrorDisplay error={error} title="Couldn't load services." />}
      {services && services.length === 0 && <EmptyRow flush>No services found in this stack yet.</EmptyRow>}
      {services && active && (
        <Tabs value={active.name} onValueChange={(name) => navigate(wsPath(`/stacks/${stackId}/logs/${name}`))}>
          <TabsList variant="line" aria-label="Service" style={edgeFade} className="relative isolate w-full justify-start overflow-x-auto overflow-y-hidden scrollbar-none border-b border-border p-0">
            <TabUnderline />
            {services.map((c) => (
              <TabsTrigger
                key={c.id}
                value={c.name}
                className="flex-none gap-2 px-3 font-mono after:hidden"
                title={c.name}
                ref={c.name === active.name ? revealActive : undefined}
              >
                <span aria-hidden className={cn("size-1.5 shrink-0 rounded-full", dotClass[c.status] ?? "bg-muted-foreground/50")} />
                <span className="max-w-56 truncate">{c.name}</span>
              </TabsTrigger>
            ))}
          </TabsList>
        </Tabs>
      )}
      {active && <LogsView key={active.name} source={stackLogSource(stackId, active.name)} />}
    </div>
  );
};
