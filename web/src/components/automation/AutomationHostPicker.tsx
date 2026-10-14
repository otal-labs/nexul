import { SettingsCard } from "@/components/settings/SettingsCard";
import { useActionState } from "@/hooks/useActionState";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { useSetAutomationHost } from "@/hooks/AutomationHooks";
import { useFetchAutomationHosts } from "@/hooks/AutomationHostHooks";
import { useFetchMachines } from "@/hooks/RunnerHooks";
import type { Automation } from "@/models/Automation";
import { canPlaceOn, InstanceHostName } from "@/models/AutomationHost";

interface AutomationHostPickerProps {
  automation: Automation;
}

// Radix Select has no empty value, so the instance host rides a sentinel that maps back to host_id null.
const instanceValue = "__instance__";

export const AutomationHostPicker = ({ automation }: AutomationHostPickerProps) => {
  const { data: hosts } = useFetchAutomationHosts();
  const { data: machines } = useFetchMachines();
  const setHost = useSetAutomationHost(automation.id);
  // Where an automation may run is checked per host against its machine; the permission here is only the workspace's.
  const action = useActionState({ permission: "automations:write", verb: "change where automations run" });
  if (action.state === "hidden") return null;
  // The host it runs on stays listed so the picker can name it, but can't be picked again without Deploy to there.
  const others = machines
    ? (hosts ?? []).filter((host) => host.name !== InstanceHostName && (canPlaceOn(host, machines) || host.id === automation.host_id))
    : [];

  return (
    <SettingsCard id="runs-on" title="Runs on" description="The automations host that runs this automation.">
      <Select
        value={automation.host_id ?? instanceValue}
        disabled={setHost.isPending || action.state !== "live"}
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
            <SelectItem key={host.id} value={host.id} disabled={!canPlaceOn(host, machines ?? [])} className="font-mono">
              {host.name}
            </SelectItem>
          ))}
        </SelectContent>
      </Select>
      {action.state === "locked" && <p className="mt-2 text-xs text-muted-foreground">{action.reason}</p>}
    </SettingsCard>
  );
};
