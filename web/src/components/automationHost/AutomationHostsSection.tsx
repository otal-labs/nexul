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
      description="Machines whose worker runs this workspace's automations, beside the one built into the instance."
      footer={<AddAutomationHostDialog />}
    >
      {isPending && <LoadingDisplay />}
      {error && <ErrorDisplay error={error} />}
      {hosts && hosts.length === 0 && <EmptyRow>No automations host enrolled yet.</EmptyRow>}
      {hosts && hosts.length > 0 && (
        <ul className="divide-y divide-border overflow-hidden rounded-md border border-border">
          {hosts.map((host, index) => (
            <AutomationHostRow key={host.id} host={host} index={index} />
          ))}
        </ul>
      )}
    </SettingsCard>
  );
};
