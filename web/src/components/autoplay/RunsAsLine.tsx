import { personLabel } from "@nexul/client-core/person";
import { useFormContext, useWatch } from "react-hook-form";

import { Button } from "@/components/ui/button";
import { useFetchMe } from "@/hooks/AuthHooks";
import { usePerson } from "@/hooks/PeopleHooks";
import type { AutoPlayDraft } from "@/models/AutoPlay";

// Whose computers a person auto play lands on once saved, and the way to make that you.
export const RunsAsLine = () => {
  const { setValue, formState } = useFormContext<AutoPlayDraft>();
  const [run_as, run_as_me] = useWatch<AutoPlayDraft, ["run_as", "run_as_me"]>({ name: ["run_as", "run_as_me"] });
  const { data: me } = useFetchMe();
  const person = usePerson(run_as);
  const yours = run_as_me || (me !== undefined && run_as === me.user.id);
  const text = (() => {
    if (yours) return "Runs as you, on your computers in your order.";
    if (run_as === "") return "Nobody picked yet.";
    return `Runs as ${personLabel(person)}, on their computers.`;
  })();

  return (
    me && (
      <div className="flex items-center gap-2">
        <p className="text-xs text-muted-foreground">{text}</p>
        {!yours && (
          <Button type="button" variant="outline" size="sm" disabled={formState.disabled} onClick={() => setValue("run_as_me", true, { shouldDirty: true })}>
            Run as me
          </Button>
        )}
      </div>
    )
  );
};
