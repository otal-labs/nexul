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
    <section className="space-y-3 rounded-lg border border-border bg-card p-4">
      <div className="space-y-1">
        <h2 className="text-sm font-semibold">
          <label htmlFor="automation-host">Runs on</label>
        </h2>
        <p className="text-xs text-muted-foreground">The automations host whose worker runs this automation.</p>
      </div>
      <Select
        value={automation.host_id ?? instanceValue}
        disabled={setHost.isPending}
        onValueChange={(value) => setHost.mutate(value === instanceValue ? null : value)}
      >
        <SelectTrigger id="automation-host" className="w-full font-mono sm:w-72">
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
    </section>
  );
};
