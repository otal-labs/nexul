import { SettingsHint } from "@/components/settings/SettingsHint";
import { useListOrderComputers } from "@/hooks/PairingHooks";
import { sharedComputerName } from "@/models/SharedComputer";

// What the defaults use for where runs go: your order, read-only here because the list beside it is where it changes.
export const DefaultsOrderList = () => {
  const { data: computers } = useListOrderComputers();
  if (!computers || computers.length === 0) return null;

  return (
    <div className="space-y-1.5 pt-2">
      <ol aria-label="Your order" className="divide-y divide-border rounded-md border border-border text-sm">
        {computers.map((computer, i) => (
          <li key={computer.id} className="flex items-baseline gap-3 px-3 py-1.5">
            <span className="w-4 font-mono text-xs text-muted-foreground tabular-nums">{i + 1}</span>
            <span className="min-w-0 truncate">{sharedComputerName(computer)}</span>
          </li>
        ))}
      </ol>
      <SettingsHint>Drag in the list to change</SettingsHint>
    </div>
  );
};
