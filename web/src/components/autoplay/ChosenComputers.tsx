import { useFormContext, useWatch } from "react-hook-form";

import { RunsAsLine } from "@/components/autoplay/RunsAsLine";
import { EmptyRow } from "@/components/EmptyRow";
import { ComputerLoad } from "@/components/settings/ComputerLoad";
import { RankedComputerRow } from "@/components/settings/RankedComputerRow";
import { RankedComputersField } from "@/components/settings/RankedComputersField";
import { useListSharedComputers } from "@/hooks/ComputerHooks";
import { useListComputers } from "@/hooks/PairingHooks";
import type { AutoPlayDraft } from "@/models/AutoPlay";
import { notRunnableYet, sharedComputerName } from "@/models/SharedComputer";

// The viewer's own computers: in their own order, or the ones they picked, ranked. One shared with them is listed as every other picker lists it, not pickable. A computer already in the list that they can't see stays a row, so a save never drops it.
export const ChosenComputers = () => {
  const { setValue, formState } = useFormContext<AutoPlayDraft>();
  const ids = useWatch<AutoPlayDraft, "computer_ids">({ name: "computer_ids" });
  const { data: own } = useListComputers();
  const { data: shared = [] } = useListSharedComputers();
  const unavailable = shared.map((computer) => ({ id: computer.id, label: `${sharedComputerName(computer)} · ${notRunnableYet}` }));
  const nameOf = (id: string) => {
    const computer = [...(own ?? []), ...shared].find((c) => c.id === id);
    return computer ? sharedComputerName(computer) : "A computer you can't see";
  };

  return (
    <div className="max-w-xl space-y-1.5">
      {own && own.length === 0 && ids.length === 0 && <EmptyRow flush>You have no computer to pick yet. Add one under Computers.</EmptyRow>}
      {own && (own.length > 0 || ids.length > 0) && (
        <RankedComputersField
          subject="auto play"
          toggle={false}
          own={own}
          ids={ids}
          onChange={(next) => setValue("computer_ids", next, { shouldDirty: true })}
          disabled={formState.disabled}
          unavailable={unavailable}
          renderRow={({ id, rank, controls }) => {
            const computer = own.find((c) => c.id === id);
            return (
              <RankedComputerRow id={id} name={nameOf(id)} rank={rank} controls={controls}>
                {computer && <ComputerLoad computer={computer} />}
              </RankedComputerRow>
            );
          }}
        />
      )}
      <RunsAsLine />
    </div>
  );
};
