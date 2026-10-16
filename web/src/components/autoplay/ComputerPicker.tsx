import { useFormContext, useWatch } from "react-hook-form";

import { InlineSelect } from "@/components/autoplay/ComposerControls";
import { EmptyRow } from "@/components/EmptyRow";
import { useListSharedComputers } from "@/hooks/ComputerHooks";
import { useListComputers } from "@/hooks/PairingHooks";
import type { AutoPlayDraft } from "@/models/AutoPlay";
import { notRunnableYet, sharedComputerName } from "@/models/SharedComputer";

// The viewer's own computers; one shared with them is listed as every other picker lists it, not pickable. A computer already picked that they can't see stays listed, so a save never drops it.
export const ComputerPicker = () => {
  const { setValue, formState } = useFormContext<AutoPlayDraft>();
  const computerId = useWatch<AutoPlayDraft, "computer_id">({ name: "computer_id" });
  const { data: computers, isPending } = useListComputers();
  const { data: shared } = useListSharedComputers();
  const own = (computers ?? []).map((computer) => ({ value: computer.id, label: computer.name }));
  const sharedOptions = (shared ?? []).map((computer) => ({ value: computer.id, label: `${sharedComputerName(computer)} · ${notRunnableYet}`, disabled: true }));
  const options = [...own, ...sharedOptions];
  const unseen = computerId !== "" && !options.some((option) => option.value === computerId);

  return (
    <div className="space-y-1.5">
      {!isPending && options.length === 0 && !unseen && (
        <EmptyRow flush>You have no computer to pick yet. Add one under Computers.</EmptyRow>
      )}
      {(options.length > 0 || unseen) && (
        <InlineSelect
          label="Computer"
          value={computerId}
          placeholder="Pick a computer"
          options={unseen ? [...options, { value: computerId, label: "A computer you can't see" }] : options}
          onChange={(value) => setValue("computer_id", value, { shouldDirty: true })}
          disabled={formState.disabled}
        />
      )}
      <p className="text-xs text-muted-foreground">The run takes the identity of whoever picks the computer.</p>
    </div>
  );
};
