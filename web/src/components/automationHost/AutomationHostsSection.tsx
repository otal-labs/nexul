import { ServerIcon } from "lucide-react";

import { AddAutomationHostDialog } from "@/components/automationHost/AddAutomationHostDialog";
import { AutomationHostRow } from "@/components/automationHost/AutomationHostRow";
import { ErrorDisplay } from "@/components/ErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { NoDataDisplay } from "@/components/NoDataDisplay";
import { useFetchAutomationHosts } from "@/hooks/AutomationHostHooks";

export const AutomationHostsSection = () => {
  const { data: hosts, error, isPending } = useFetchAutomationHosts();

  return (
    <section className="space-y-3">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <h2 className="flex items-center gap-2 font-mono text-xs uppercase tracking-[0.2em] text-muted-foreground">
          <ServerIcon className="size-4" aria-hidden />
          Automations hosts
        </h2>
        <AddAutomationHostDialog />
      </div>
      {isPending && <LoadingDisplay />}
      {error && <ErrorDisplay error={error} />}
      {hosts && hosts.length === 0 && <NoDataDisplay size="compact" message="No automations host enrolled yet." />}
      {hosts && hosts.length > 0 && (
        <ul className="divide-y divide-border rounded-lg border border-border bg-card">
          {hosts.map((host, index) => (
            <AutomationHostRow key={host.id} host={host} index={index} />
          ))}
        </ul>
      )}
    </section>
  );
};
