import { useFormContext, useWatch } from "react-hook-form";

import { EmptyRow } from "@/components/EmptyRow";
import { RankedComputerRow } from "@/components/settings/RankedComputerRow";
import { RankedComputersField } from "@/components/settings/RankedComputersField";
import { useListSharedComputers } from "@/hooks/ComputerHooks";
import { useListComputers } from "@/hooks/PairingHooks";
import type { AutoPlayDraft } from "@/models/AutoPlay";
import { notRunnableYet, sharedComputerName } from "@/models/SharedComputer";

// The viewer's own computers, ranked; one shared with them is listed as every other picker lists it, not pickable. A computer already in the list that they can't see stays a row, so a save never drops it.
export const ChosenComputers = () => {
  const { setValue, formState } = useFormContext<AutoPlayDraft>();
  const ids = useWatch<AutoPlayDraft, "computer_ids">({ name: "computer_ids" });
  const picked = useWatch<AutoPlayDraft, "computers_picked">({ name: "computers_picked" });
  const { data: own } = useListComputers();
  const { data: shared = [] } = useListSharedComputers();
  const unavailable = shared.map((computer) => ({ id: computer.id, label: `${sharedComputerName(computer)} · ${notRunnableYet}` }));
  const nameOf = (id: string) => {
    const computer = [...(own ?? []), ...shared].find((c) => c.id === id);
    return computer ? sharedComputerName(computer) : "A computer you can't see";
  };
  const choose = (next: string[]) => {
    setValue("computer_ids", next, { shouldDirty: true });
    setValue("computers_picked", true, { shouldDirty: true });
  };

  return (
    <div className="max-w-xl space-y-1.5">
      {own && own.length === 0 && ids.length === 0 && <EmptyRow flush>You have no computer to pick yet. Add one under Computers.</EmptyRow>}
      {own && (own.length > 0 || ids.length > 0) && (
        <RankedComputersField
          subject="auto play"
          own={own}
          ids={ids}
          chosen={picked}
          onChange={choose}
          disabled={formState.disabled}
          unavailable={unavailable}
          renderRow={({ id, rank, actions }) => <RankedComputerRow name={nameOf(id)} rank={rank} actions={actions} />}
        />
      )}
      <p className="text-xs text-muted-foreground">The run takes the identity of whoever picks the computers.</p>
    </div>
  );
};
