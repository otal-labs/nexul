import { useEffect } from "react";
import { useForm, useWatch } from "react-hook-form";

import { Button } from "@/components/ui/button";
import { ErrorDisplay } from "@/components/ErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { NoComputersDisplay } from "@/components/play/NoComputersDisplay";
import { microheaderClass } from "@/components/Microheader";
import { HarnessComputerField } from "@/components/settings/HarnessComputerField";
import { HarnessProjectField } from "@/components/settings/HarnessProjectField";
import { useFetchPresence, useListComputers } from "@/hooks/PairingHooks";
import { useHarnessPlaceNames } from "@/hooks/PairingProjectHooks";
import { cn } from "@/lib/utils";
import { passOverLines, type PassOver } from "@/models/Pairing";

// Where a run goes: one of the person's computers and a T3 project on it (ADR 0165).
export interface RunWhere {
  computer_id: string;
  harness_project_id: string;
}

interface RunWhereSectionProps {
  value: RunWhere;
  // The computers the run would pass over before the one it lands on, and why.
  passedOver: PassOver[];
  // Open pickers instead of the one-line summary: no computer has the project, a Change, or a refusal that asked where.
  asking: boolean;
  mustPick: boolean;
  onChange: (next: RunWhere) => void;
  onChangeRequested: () => void;
}

type RunWhereFormProps = Pick<RunWhereSectionProps, "value" | "onChange">;

// Subscribe so the run payload always matches the fields displayed in this form.
const RunWhereForm = ({ value, onChange }: RunWhereFormProps) => {
  const { data: computers, isPending, error } = useListComputers();
  const { data: presence } = useFetchPresence();
  const form = useForm<RunWhere>({ defaultValues: value });
  const computerId = useWatch({ control: form.control, name: "computer_id" });

  useEffect(() => form.subscribe({ formState: { values: true }, callback: ({ values }) => onChange(values) }), [form, onChange]);

  return (
    <div className="space-y-3">
      {isPending && <LoadingDisplay label="Loading computers" className="justify-start p-0" />}
      {error && <ErrorDisplay error={error} title="Couldn't load your computers." />}
      {computers && computers.length === 0 && <NoComputersDisplay />}
      {computers && computers.length > 0 && (
        <HarnessComputerField
          control={form.control}
          name="computer_id"
          computers={computers}
          presence={presence ?? {}}
          onChangeValue={() => form.setValue("harness_project_id", "")}
          withShared
        />
      )}
      <HarnessProjectField control={form.control} name="harness_project_id" label="T3 project" computerId={computerId} />
    </div>
  );
};

export const RunWhereSection = ({ value, passedOver, asking, mustPick, onChange, onChangeRequested }: RunWhereSectionProps) => {
  const { computer, project, isPending } = useHarnessPlaceNames(value.computer_id, value.harness_project_id);
  const note = mustPick
    ? "None of your computers has this project yet. Pick a computer and its T3 project; it's remembered there for this project."
    : "This run goes here. Your order of computers stays as it is.";

  return (
    <section className="space-y-2">
      <h3 className={microheaderClass}>Where to run</h3>
      {!asking && isPending && <LoadingDisplay label="Loading where it runs" className="justify-start p-0" />}
      {!asking && !isPending && (
        <div className="flex items-center gap-3">
          <p className="min-w-0 flex-1 truncate font-mono text-xs text-muted-foreground">
            {computer} · {project}
          </p>
          <Button type="button" variant="ghost" size="sm" onClick={onChangeRequested}>
            Change
          </Button>
        </div>
      )}
      {!asking && passedOver.length > 0 && <p className="text-xs text-muted-foreground">Skipping: {passOverLines(passedOver)}.</p>}
      {asking && (
        <div className={cn("space-y-3", !mustPick && "settle-in")}>
          <p className="text-xs text-muted-foreground">{note}</p>
          <RunWhereForm value={value} onChange={onChange} />
        </div>
      )}
    </section>
  );
};
