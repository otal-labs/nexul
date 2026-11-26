import { EnterList } from "@/components/EnterList";
import { microheaderClass } from "@/components/Microheader";
import { InboxEntryRow } from "@/components/notifications/InboxEntryRow";
import type { InboxDayGroup } from "@/utils/InboxUtility";
import { cn } from "@/lib/utils";

// One day band of the inbox under its pinned microheader; news arriving rises in at the top of Today.
export const InboxDaySection = ({ group }: { group: InboxDayGroup }) => (
  <section aria-label={group.label}>
    <h2 className={cn(microheaderClass, "sticky-group-head flex items-end px-3 pb-1.5")}>{group.label}</h2>
    <EnterList as="div" arrival="rise">
      {group.entries.map((e) => <InboxEntryRow key={e.key} entry={e} />)}
    </EnterList>
  </section>
);
