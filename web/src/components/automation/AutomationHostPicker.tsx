import { SettingsCard } from "@/components/settings/SettingsCard";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { useSetAutomationHost } from "@/hooks/AutomationHooks";
import { useFetchAutomationHosts } from "@/hooks/AutomationHostHooks";
import type { Automation } from "@/models/Automation";
import { InstanceHostName } from "@/models/AutomationHost";

interface AutomationHostPickerProps {
  automation: Automation;
}

// Radix Select has no empty value, so the instance host rides a sentinel that maps back to host_id null.
const instanceValue = "__instance__";

export const AutomationHostPicker = ({ automation }: AutomationHostPickerProps) => {
  const { data: hosts } = useFetchAutomationHosts();
  const setHost = useSetAutomationHost(automation.id);
  const others = (hosts ?? []).filter((host) => host.name !== InstanceHostName);

  return (
    <SettingsCard id="runs-on" title="Runs on" description="The automations host that runs this automation.">
      <Select
        value={automation.host_id ?? instanceValue}
        disabled={setHost.isPending}
        onValueChange={(value) => setHost.mutate(value === instanceValue ? null : value)}
      >
        <SelectTrigger aria-labelledby="runs-on-title" className="w-full font-mono sm:w-72">
          <SelectValue />
        </SelectTrigger>
        <SelectContent>
          <SelectItem value={instanceValue} className="font-mono">
            instance (built in)
          </SelectItem>
          {others.map((host) => (
            <SelectItem key={host.id} value={host.id} className="font-mono">
              {host.name}
            </SelectItem>
          ))}
        </SelectContent>
      </Select>
    </SettingsCard>
  );
};
