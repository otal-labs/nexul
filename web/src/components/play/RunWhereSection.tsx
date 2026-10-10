import { useEffect } from "react";
import { useForm, useWatch } from "react-hook-form";

import { Button } from "@/components/ui/button";
import { ErrorDisplay } from "@/components/ErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { NoDataDisplay } from "@/components/NoDataDisplay";
import { microheaderClass } from "@/components/Microheader";
import { HarnessComputerField } from "@/components/settings/HarnessComputerField";
import { HarnessProjectField } from "@/components/settings/HarnessProjectField";
import { useFetchHarnessProjects, useFetchPresence, useListComputers } from "@/hooks/PairingHooks";
import { cn } from "@/lib/utils";

// Where a run goes: one of the person's computers and a T3 project on it, saved as their project link (ADR 0145).
export interface RunWhere {
  computer_id: string;
  harness_project_id: string;
}

interface RunWhereSectionProps {
  value: RunWhere;
  // Open pickers instead of the one-line summary: a first run, a Change, or a refusal that asked where.
  asking: boolean;
  firstRun: boolean;
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
      {computers && computers.length === 0 && <NoDataDisplay message="Add a computer in Settings to run plays." size="compact" />}
      {computers && computers.length > 0 && (
        <HarnessComputerField
          control={form.control}
          name="computer_id"
          computers={computers}
          presence={presence ?? {}}
          onChangeValue={() => form.setValue("harness_project_id", "")}
        />
      )}
      <HarnessProjectField control={form.control} name="harness_project_id" label="T3 project" computerId={computerId} />
    </div>
  );
};

export const RunWhereSection = ({ value, asking, firstRun, onChange, onChangeRequested }: RunWhereSectionProps) => {
  const { data: computers } = useListComputers();
  const { data: projects } = useFetchHarnessProjects(value.computer_id);
  const computer = computers?.find((c) => c.id === value.computer_id)?.name ?? value.computer_id;
  const project = projects?.find((p) => p.id === value.harness_project_id)?.title ?? value.harness_project_id;
  const note = firstRun
    ? "Your first run in this project. Pick where it runs and it's saved as your link here, so later runs skip this."
    : "What you pick here replaces your link for this project.";

  return (
    <section className="space-y-2">
      <h3 className={microheaderClass}>Where to run</h3>
      {!asking && (
        <div className="flex items-center gap-3">
          <p className="min-w-0 flex-1 truncate font-mono text-xs text-muted-foreground">
            {computer} · {project}
          </p>
          <Button type="button" variant="ghost" size="sm" onClick={onChangeRequested}>
            Change
          </Button>
        </div>
      )}
      {asking && (
        <div className={cn("space-y-3", !firstRun && "settle-in")}>
          <p className="text-xs text-muted-foreground">{note}</p>
          <RunWhereForm value={value} onChange={onChange} />
        </div>
      )}
    </section>
  );
};
