import { zodResolver } from "@hookform/resolvers/zod";
import { useState } from "react";
import { Controller, useForm } from "react-hook-form";
import { PlusIcon } from "lucide-react";

import { FormInput } from "@/components/FormInput";
import { StatusIconPicker } from "@/components/settings/StatusIconPicker";
import { StatusRow } from "@/components/settings/StatusRow";
import { Button } from "@/components/ui/button";
import { useCreateStatus, useReorderStatuses } from "@/hooks/StatusHooks";
import { SaveStatusFormSchema, STATUS_STAGES, StatusKind, type SaveStatusFormData } from "@/models/Status";
import type { BoardStatus } from "@/models/Status";

interface BoardStatusColumnsSectionProps {
  projectId: string;
  statuses: BoardStatus[];
}

export const BoardStatusColumnsSection = ({ projectId, statuses }: BoardStatusColumnsSectionProps) => {
  const createStatus = useCreateStatus();
  const reorderStatuses = useReorderStatuses();
  const [addingKind, setAddingKind] = useState<StatusKind | null>(null);

  const statusForm = useForm<SaveStatusFormData>({
    defaultValues: { name: "", kind: StatusKind.Backlog, icon: "" },
    resolver: zodResolver(SaveStatusFormSchema),
  });

  const moveStatus = (index: number, direction: -1 | 1) => {
    const ids = statuses.map((s) => s.id);
    if (!ids.length) return;
    const target = index + direction;
    const from = ids[index];
    const to = ids[target];
    if (target < 0 || target >= ids.length || from === undefined || to === undefined) return;
    ids[index] = to;
    ids[target] = from;
    void reorderStatuses.mutateAsync({ project_id: projectId, ids });
  };

  // Each group's own "+" presets kind — no separate selector; clicking again closes it.
  const toggleAdding = (kind: StatusKind) => {
    if (addingKind === kind) {
      setAddingKind(null);
      return;
    }
    statusForm.reset({ name: "", kind, icon: "" });
    setAddingKind(kind);
  };

  return (
    <div>
      <h3 className="text-sm font-semibold">Status columns</h3>
      <div className="mt-3 space-y-4">
        {STATUS_STAGES.map(({ kind, label }) => {
          const rows = statuses
            .map((status, index) => ({ status, index }))
            .filter(({ status }) => status.kind === kind);

          return (
            <div key={kind}>
              <div className="flex items-center justify-between gap-2 border-b border-border pb-1.5">
                <span className="text-xs font-medium tracking-wide text-muted-foreground uppercase">
                  {label}
                </span>
                <Button
                  variant="ghost"
                  size="icon"
                  className="size-6"
                  aria-label={`Add ${kind} status`}
                  onClick={() => toggleAdding(kind)}
                >
                  <PlusIcon className="size-3.5" />
                </Button>
              </div>
              {rows.length === 0 && (
                <p className="py-2 text-sm text-muted-foreground">No {kind} statuses — skipped on this board</p>
              )}
              {rows.length > 0 && (
                // Named for assistive tech since the header states kind once and rows don't repeat it.
                <ul aria-label={`${label} statuses`} className="divide-y divide-border">
                  {rows.map(({ status, index }) => (
                    <StatusRow
                      key={status.id}
                      status={status}
                      index={index}
                      total={statuses.length}
                      onMove={(direction) => moveStatus(index, direction)}
                    />
                  ))}
                </ul>
              )}
              {addingKind === kind && (
                <form
                  className="mt-2 flex flex-wrap items-end gap-2"
                  onSubmit={statusForm.handleSubmit(async (data) => {
                    await createStatus.mutateAsync({ ...data, project_id: projectId });
                    statusForm.reset({ name: "", kind: StatusKind.Backlog, icon: "" });
                    setAddingKind(null);
                  })}
                >
                  <FormInput
                    control={statusForm.control}
                    name="name"
                    id="new-status-name"
                    label="New status name"
                    placeholder="New status name"
                    hideLabel
                    className="w-full sm:w-48"
                  />
                  <Controller
                    control={statusForm.control}
                    name="icon"
                    render={({ field }) => (
                      <StatusIconPicker label="New status icon" value={field.value} onChange={field.onChange} />
                    )}
                  />
                  <Button
                    type="submit"
                    size="icon"
                    variant="outline"
                    aria-label="Add column"
                    disabled={statusForm.formState.isSubmitting}
                  >
                    <PlusIcon className="size-4" />
                  </Button>
                </form>
              )}
            </div>
          );
        })}
      </div>
    </div>
  );
};
