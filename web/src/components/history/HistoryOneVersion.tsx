import { personLabel } from "@nexul/client-core/person";

import { footerTrayClass } from "@/components/history/HistoryFooter";
import { Button } from "@/components/ui/button";
import { DialogClose } from "@/components/ui/dialog";
import { usePersonLookup } from "@/hooks/PeopleHooks";
import type { HistoryVersionSummary } from "@/models/History";
import { useWorkspaceStore } from "@/stores/workspaceStore";
import { viaLabel } from "@/utils/HistoryVersionUtility";
import { formatVersionTime } from "@/utils/TimeUtility";

interface HistoryOneVersionProps {
  version: HistoryVersionSummary;
}

// A single version has nothing to compare or restore, so the dialog shrinks to what there is to say, with Close alone.
export const HistoryOneVersion = ({ version }: HistoryOneVersionProps) => {
  const person = usePersonLookup(useWorkspaceStore((s) => s.selectedWorkspaceId));
  const people = version.editor_ids.map((id) => personLabel(person(id)));
  const by = people.length > 0 ? ` by ${people.join(", ")}` : "";
  const via = version.via ? ` ${viaLabel(version.via)}` : "";

  return (
    <>
      <p className="bg-popover px-6 pb-5 text-[13px] text-muted-foreground">
        <span className="text-foreground">
          One version, saved {formatVersionTime(version.created_at)}
          {by}
          {via}.
        </span>{" "}
        Older versions show here after the next edit.
      </p>
      <div className={`${footerTrayClass} justify-end`}>
        <DialogClose asChild>
          <Button variant="ghost">Close</Button>
        </DialogClose>
      </div>
    </>
  );
};
