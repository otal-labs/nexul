import { EnterList } from "@/components/EnterList";
import { AddAutomationHostDialog } from "@/components/automationHost/AddAutomationHostDialog";
import { AutomationHostRow } from "@/components/automationHost/AutomationHostRow";
import { EmptyRow } from "@/components/EmptyRow";
import { ErrorDisplay } from "@/components/ErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { SettingsCard } from "@/components/settings/SettingsCard";
import { useFetchAutomationHosts } from "@/hooks/AutomationHostHooks";

export const AutomationHostsSection = () => {
  const { data: hosts, error, isPending } = useFetchAutomationHosts();

  return (
    <SettingsCard
      id="automation-hosts"
      title="Automations hosts"
      description="Where this workspace's automations run, besides the instance's own host."
      footer={<AddAutomationHostDialog />}
    >
      {isPending && <LoadingDisplay />}
      {error && <ErrorDisplay error={error} />}
      {hosts && hosts.length === 0 && <EmptyRow>No automations hosts yet. Add one to run automations on another machine.</EmptyRow>}
      {hosts && hosts.length > 0 && (
        <EnterList className="divide-y divide-border overflow-hidden rounded-md border border-border">
          {hosts.map((host) => (
            <AutomationHostRow key={host.id} host={host} />
          ))}
        </EnterList>
      )}
    </SettingsCard>
  );
};
