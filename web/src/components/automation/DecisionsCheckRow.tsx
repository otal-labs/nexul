import { AutomationKindBadge } from "@/components/automation/AutomationKindBadge";
import { Switch } from "@/components/ui/switch";
import { AutomationKind } from "@/enums/Automation";
import { useFetchDecisionsCheck, useSetDecisionsCheckEnabled } from "@/hooks/PlayHooks";

// The built-in decisions check sits among the defaults; it has no detail page, only its switch.
export const DecisionsCheckRow = () => {
  const { data: check } = useFetchDecisionsCheck();
  const setEnabled = useSetDecisionsCheckEnabled();

  if (!check) return null;
  return (
    <li className="flex items-center gap-3 p-4 transition-colors duration-150 ease-standard hover:bg-accent/40">
        <div className="min-w-0 flex-1">
          <div className="flex flex-wrap items-center gap-2">
            <span className="truncate text-sm font-medium">{check.label}</span>
            <AutomationKindBadge kind={AutomationKind.Default} />
          </div>
          <p className="truncate text-sm text-muted-foreground">{check.description}</p>
        </div>
        <Switch
          aria-label={check.enabled ? `Disable ${check.label}` : `Enable ${check.label}`}
          checked={check.enabled}
          disabled={setEnabled.isPending}
          onCheckedChange={(checked) => setEnabled.mutate(checked)}
        />
    </li>
  );
};
