import { EnterList } from "@/components/EnterList";
import { Microheader } from "@/components/Microheader";
import { AutomationRow } from "@/components/automation/AutomationRow";
import type { AutomationGroup } from "@/utils/AutomationListUtility";

interface AutomationGroupSectionProps {
  group: AutomationGroup;
}

export const AutomationGroupSection = ({ group }: AutomationGroupSectionProps) => (
  <section aria-label={group.label}>
    <Microheader className="px-4 pt-5 pb-2">{group.label}</Microheader>
    <EnterList className="divide-y divide-border border-b border-border">
      {group.items.map((item) => (
        <AutomationRow key={item.automation.id} item={item} />
      ))}
    </EnterList>
  </section>
);
