import { personLabel } from "@nexul/client-core/person";

import { HistorySpine } from "@/components/history/HistorySpine";
import { usePersonLookup } from "@/hooks/PeopleHooks";
import type { HistoryFieldChange } from "@/models/History";
import { useWorkspaceStore } from "@/stores/workspaceStore";
import { formatFullTime } from "@/utils/TimeUtility";

// "due_on" reads "Due on".
const fieldName = (field: string) => field.charAt(0).toUpperCase() + field.slice(1).replaceAll("_", " ");

interface HistoryFieldRowProps {
  change: HistoryFieldChange;
  first: boolean;
  last: boolean;
}

// A field a save changed, between the versions it happened among: one muted line, "Status  In progress → Testing". It is
// not part of any description version, so it is never picked; the person and the time are in its tooltip.
export const HistoryFieldRow = ({ change, first, last }: HistoryFieldRowProps) => {
  const person = usePersonLookup(useWorkspaceStore((s) => s.selectedWorkspaceId));
  const { user_id: userId, automation_name: automation, play_label: play } = change.actor;
  const who = (userId && personLabel(person(userId))) || automation || play || "";

  return (
    <div aria-disabled="true" title={[who, formatFullTime(change.created_at)].filter(Boolean).join(" · ")} className="flex h-7 items-center gap-2 pr-4 pl-4 text-xs">
      <HistorySpine node="field" first={first} last={last} />
      <span className="min-w-0 flex-1 truncate">
        <span className="mr-2 text-muted-foreground">{fieldName(change.field)}</span>
        {change.from && <span className="text-muted-foreground">{change.from} → </span>}
        {change.to}
      </span>
    </div>
  );
};
