import { ServerIcon } from "lucide-react";

import { EnterList } from "@/components/EnterList";
import { AutomationListBar } from "@/components/automation/AutomationListBar";
import { AddAutomationHostDialog } from "@/components/automationHost/AddAutomationHostDialog";
import { AutomationHostRow } from "@/components/automationHost/AutomationHostRow";
import { EmptyState } from "@/components/EmptyState";
import { ErrorDisplay } from "@/components/ErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { useFetchAutomationHosts } from "@/hooks/AutomationHostHooks";
import { useFetchMachines } from "@/hooks/RunnerHooks";
import { useActionState } from "@/hooks/useActionState";
import { useAutomationHostManagement } from "@/hooks/useAutomationHostManagement";
import { InstanceHostName, machineOfHost } from "@/models/AutomationHost";

export const AutomationHostsSection = () => {
  const { data: hosts, error, isPending } = useFetchAutomationHosts();
  const { data: machines } = useFetchMachines();
  const { answer } = useAutomationHostManagement();
  const add = useActionState({ permission: "machines:write", answer, verb: "add automations hosts" });
  const instanceOnly = hosts?.length === 1 && hosts[0]?.name === InstanceHostName;
  const addHost = add.state !== "hidden" && <AddAutomationHostDialog />;

  return (
    <section aria-label="Hosts" className="space-y-1">
      <AutomationListBar
        summary={
          <>
            {hosts && hosts.length > 0 && (
              <span className="font-mono tabular-nums">
                {hosts.length} {hosts.length === 1 ? "host" : "hosts"} ·{" "}
              </span>
            )}
            Machines that run this workspace&apos;s automations.
          </>
        }
      >
        {hosts?.length !== 0 && addHost}
      </AutomationListBar>
      {isPending && <LoadingDisplay />}
      {error && <ErrorDisplay error={error} />}
      {hosts && hosts.length === 0 && (
        <EmptyState icon={ServerIcon} title="No hosts yet" message="Every automation runs on this instance until you add one." action={addHost} />
      )}
      {hosts && hosts.length > 0 && (
        <EnterList className="divide-y divide-border border-y border-border">
          {hosts.map((host) => (
            <AutomationHostRow key={host.id} host={host} machine={machineOfHost(host, machines)} />
          ))}
        </EnterList>
      )}
      {instanceOnly && <p className="px-4 pt-2 text-xs text-muted-foreground">Add a host to run automations on a machine of your own, beside the secrets manager they read.</p>}
    </section>
  );
};
