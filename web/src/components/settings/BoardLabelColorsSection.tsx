import { useState } from "react";
import { zodResolver } from "@hookform/resolvers/zod";
import { PlusIcon } from "lucide-react";
import { useForm } from "react-hook-form";

import { ActionButton } from "@/components/action/ActionButton";
import { CONFIGURABLE_COLOR_NAMES } from "@/components/board/ticketTypeColor";
import { EnterList } from "@/components/EnterList";
import { EmptyRow } from "@/components/EmptyRow";
import { ErrorDisplay } from "@/components/ErrorDisplay";
import { FormInput } from "@/components/FormInput";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { LabelColorRow } from "@/components/settings/LabelColorRow";
import { SettingsCard } from "@/components/settings/SettingsCard";
import { settingsRowListClass } from "@/components/settings/SettingsRow";
import { Button } from "@/components/ui/button";
import { useCreateLabel, useFetchAllLabels, useFetchLabelColors } from "@/hooks/LabelHooks";
import { SaveLabelFormSchema, type SaveLabelFormData } from "@/models/Ticket";

interface BoardLabelColorsSectionProps {
  projectId: string;
}

// Labels are text on tickets (ADR 0005); one made here is its color row until a ticket carries it.
export const BoardLabelColorsSection = ({ projectId }: BoardLabelColorsSectionProps) => {
  const { data: allLabels, isPending, error } = useFetchAllLabels(projectId);
  const { data: labelColors } = useFetchLabelColors(projectId);
  const createLabel = useCreateLabel();
  const [adding, setAdding] = useState(false);

  const labelForm = useForm<SaveLabelFormData>({
    defaultValues: { name: "" },
    resolver: zodResolver(SaveLabelFormSchema),
  });

  return (
    <SettingsCard
      id="labels"
      title="Labels"
      description="Make labels here or on a ticket. Pick the color each one shows on this board."
      footer={
        !adding && (
          <ActionButton permission="tickets:write" projectId={projectId} verb="add labels" variant="outline" size="sm" icon={PlusIcon} onClick={() => setAdding(true)}>
            New label
          </ActionButton>
        )
      }
    >
      {isPending && <LoadingDisplay />}
      {error && <ErrorDisplay error={error} />}
      {allLabels && allLabels.length === 0 && !adding && <EmptyRow>No labels yet. Add one here or on a ticket.</EmptyRow>}
      {allLabels && allLabels.length > 0 && (
        <EnterList className={settingsRowListClass}>
          {allLabels.map((label) => (
            <LabelColorRow key={label} label={label} color={labelColors?.[label] ?? ""} projectId={projectId} />
          ))}
        </EnterList>
      )}
      {adding && (
        <form
          className="settle-in mt-4 flex flex-wrap items-end gap-2"
          onSubmit={labelForm.handleSubmit(async ({ name }) => {
            if (allLabels?.includes(name)) {
              labelForm.setError("name", { message: "That label already exists" });
              return;
            }
            const color = CONFIGURABLE_COLOR_NAMES[(allLabels?.length ?? 0) % CONFIGURABLE_COLOR_NAMES.length] ?? "cyan";
            await createLabel.mutateAsync({ label: name, color, project_id: projectId });
            labelForm.reset({ name: "" });
            setAdding(false);
          })}
        >
          <FormInput control={labelForm.control} name="name" id="new-label-name" label="New label" placeholder="e.g. urgent, blocked" className="w-full sm:w-80" autoFocus />
          <Button type="submit" loading={labelForm.formState.isSubmitting}>
            Add label
          </Button>
          <Button type="button" variant="ghost" onClick={() => setAdding(false)}>
            Cancel
          </Button>
        </form>
      )}
    </SettingsCard>
  );
};
