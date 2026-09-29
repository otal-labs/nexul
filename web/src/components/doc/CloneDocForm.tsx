import { useEffect } from "react";

import { useFormDialogContext } from "@/components/dialogs/FormDialogContext";
import { ErrorDisplay } from "@/components/ErrorDisplay";
import { FormSelect } from "@/components/ticket/FormSelect";
import { useCloneDoc } from "@/hooks/DocHooks";
import { useFetchCloneDestinations } from "@/hooks/MemoryHooks";
import type { CloneDocFormData } from "@/models/Doc";

interface CloneDocFormProps {
  docId: string;
}

// Every project in every workspace the viewer belongs to; the server refuses one they can't write docs in.
export const CloneDocForm = ({ docId }: CloneDocFormProps) => {
  const { control, onSubmit, setLoading } = useFormDialogContext<CloneDocFormData>();
  const { data: destinations, error } = useFetchCloneDestinations();
  const cloneDoc = useCloneDoc();

  useEffect(() => {
    setLoading(!destinations);
  }, [destinations, setLoading]);

  onSubmit(async (input) => {
    const cloned = await cloneDoc.mutateAsync({ id: docId, projectId: input.project_id });
    return { project_id: cloned.project_id, id: cloned.id };
  });

  const options = (destinations ?? [])
    .flatMap((d) => d.projects.map((p) => ({ value: p.id, label: `${d.workspace.name} / ${p.name}` })))
    .sort((a, b) => a.label.localeCompare(b.label));

  return (
    <div className="space-y-2">
      {error && <ErrorDisplay error={error} />}
      <FormSelect control={control} name="project_id" label="Destination" options={options} />
      <p className="text-xs text-muted-foreground">Its own project makes a copy beside it.</p>
    </div>
  );
};
