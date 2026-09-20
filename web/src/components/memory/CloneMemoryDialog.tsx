import { zodResolver } from "@hookform/resolvers/zod";
import { useForm } from "react-hook-form";
import { useNavigate } from "react-router";
import { z } from "zod";

import { ErrorDisplay } from "@/components/ErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { FormSelect } from "@/components/ticket/FormSelect";
import { Button } from "@/components/ui/button";
import { Dialog, DialogContent, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { useCloneMemory, useFetchCloneDestinations } from "@/hooks/MemoryHooks";

const cloneMemorySchema = z.object({
  destination: z.string().trim().min(1, "A destination is required"),
});

type CloneMemoryFormData = z.infer<typeof cloneMemorySchema>;

interface CloneMemoryDialogProps {
  memoryId: string;
  open: boolean;
  onClose: () => void;
}

// destination packs "<workspaceId>::<projectId>" into one FormSelect value; projectId "" targets workspace scope.
const encodeDestination = (workspaceId: string, projectId: string) => `${workspaceId}::${projectId}`;
const decodeDestination = (destination: string) => {
  const [workspaceId = "", projectId = ""] = destination.split("::");
  return { workspaceId, projectId };
};

// Flattened, not option-grouped: the shared FormSelect has no group support, so each option's label carries
// its workspace prefix ("Engineering / Backend"); each workspace also offers a "Workspace" option for workspace scope.
export const CloneMemoryDialog = ({ memoryId, open, onClose }: CloneMemoryDialogProps) => {
  const navigate = useNavigate();
  const { data: destinations, error, isPending } = useFetchCloneDestinations();
  const clone = useCloneMemory();
  const form = useForm<CloneMemoryFormData>({
    resolver: zodResolver(cloneMemorySchema),
    defaultValues: { destination: "" },
  });

  const options = (destinations ?? []).flatMap((d) => [
    { value: encodeDestination(d.workspace.id, ""), label: `${d.workspace.name} / Workspace` },
    ...d.projects
      .map((p) => ({ value: encodeDestination(d.workspace.id, p.id), label: `${d.workspace.name} / ${p.name}` }))
      .sort((a, b) => a.label.localeCompare(b.label)),
  ]);

  const onSubmit = async (data: CloneMemoryFormData) => {
    const { workspaceId, projectId } = decodeDestination(data.destination);
    try {
      const cloned = await clone.mutateAsync({ id: memoryId, projectId, workspaceId });
      onClose();
      navigate(`/memories/${cloned.id}`);
    } catch {
      // Error is surfaced by the hook's toast.
    }
  };

  return (
    <Dialog open={open} onOpenChange={(o) => { if (!o) onClose(); }}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Clone to…</DialogTitle>
        </DialogHeader>
        {isPending && <LoadingDisplay />}
        {error && <ErrorDisplay error={error} />}
        {destinations && (
          <form id="clone-memory-form" className="space-y-4" onSubmit={form.handleSubmit(onSubmit)}>
            <FormSelect
              control={form.control}
              name="destination"
              label="Destination"
              options={options}
              placeholder="Select a destination"
            />
            <DialogFooter>
              <Button variant="outline" type="button" onClick={onClose}>
                Cancel
              </Button>
              <Button type="submit" disabled={clone.isPending}>
                {clone.isPending ? "Cloning…" : "Clone"}
              </Button>
            </DialogFooter>
          </form>
        )}
      </DialogContent>
    </Dialog>
  );
};
