import { personLabel } from "@nexul/client-core/person";

import { HistoryFieldValue } from "@/components/history/HistoryFieldValue";
import { HistorySpine } from "@/components/history/HistorySpine";
import { usePersonLookup } from "@/hooks/PeopleHooks";
import type { HistoryFieldChange } from "@/models/History";
import { useWorkspaceStore } from "@/stores/workspaceStore";
import { actorLabel, deltaSign, fieldName } from "@/utils/HistoryChangeUtility";
import { formatFullTime } from "@/utils/TimeUtility";

const shapeOf = (field: string, sign: string | undefined): "created" | "delta" | "move" => {
  if (field === "created") return "created";
  return sign ? "delta" : "move";
};

interface HistoryFieldRowProps {
  change: HistoryFieldChange;
  first: boolean;
  last: boolean;
}

// A field a save changed, between the versions it happened among: one muted line, "Status  In progress → Testing". It is
// not part of any description version, so it is never picked; who made it, through what, and when are in its tooltip and
// its accessible name.
export const HistoryFieldRow = ({ change, first, last }: HistoryFieldRowProps) => {
  const person = usePersonLookup(useWorkspaceStore((s) => s.selectedWorkspaceId));
  const { field, from, to, actor } = change;
  const who = actorLabel(actor, actor.user_id ? personLabel(person(actor.user_id)) : "");
  const when = [who, formatFullTime(change.created_at)].filter(Boolean).join(" · ");
  const sign = deltaSign(field, from, to);
  const shape = shapeOf(field, sign);

  return (
    <div aria-disabled="true" title={when} className="relative flex h-7 items-center gap-2 pr-4 pl-4 text-xs">
      <HistorySpine node="field" first={first} last={last} />
      <span className="flex min-w-0 flex-1 items-center gap-1.5">
        <span className="shrink-0 text-muted-foreground">{fieldName(field)}</span>
        {shape === "created" && <span className="min-w-0 truncate text-muted-foreground">by {who || "someone"}</span>}
        {shape === "delta" && (
          <>
            <span className="shrink-0 text-muted-foreground">{sign}</span>
            <HistoryFieldValue field={field} value={sign === "+" ? to : from} />
          </>
        )}
        {shape === "move" && (
          <>
            <HistoryFieldValue field={field} value={from} muted />
            <span className="shrink-0 text-muted-foreground">→</span>
            <HistoryFieldValue field={field} value={to} />
          </>
        )}
        <span className="sr-only">, {when}</span>
      </span>
    </div>
  );
};
