import { personLabel } from "@nexul/client-core/person";
import { useFormContext, useWatch } from "react-hook-form";

import { useFetchMe } from "@/hooks/AuthHooks";
import { usePerson } from "@/hooks/PeopleHooks";
import { runsAsAfterSave, type AutoPlayDraft } from "@/models/AutoPlay";

// Whose identity the run takes once this is saved: the viewer's, unless someone else picked these computers and they are left as they were.
export const RunsAsLine = () => {
  const { formState } = useFormContext<AutoPlayDraft>();
  const [computer_ids, run_as] = useWatch<AutoPlayDraft, ["computer_ids", "run_as"]>({ name: ["computer_ids", "run_as"] });
  const { data: me } = useFetchMe();
  const saved = (formState.defaultValues?.computer_ids ?? []).filter((id) => id !== undefined);
  const runsAs = me ? runsAsAfterSave({ run_as, computer_ids }, saved, me.user.id) : "";
  const person = usePerson(runsAs);

  return (
    me && (
      <p className="text-xs text-muted-foreground">{runsAs === me.user.id ? "Runs as you." : `Runs as ${personLabel(person)}.`}</p>
    )
  );
};
