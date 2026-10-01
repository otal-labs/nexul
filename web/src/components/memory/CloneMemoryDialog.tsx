import { zodResolver } from "@hookform/resolvers/zod";
import { useForm } from "react-hook-form";
import { useNavigate } from "react-router";
import { z } from "zod";

import { ErrorDisplay } from "@/components/ErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { FormSelect } from "@/components/ticket/FormSelect";
import { Button } from "@/components/ui/button";
import { Dialog, DialogContent, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { useCloneMemory, useFetchCloneDestinations, type CloneDestination } from "@/hooks/MemoryHooks";
import { memoryPath, projectToken } from "@/models/Project";
import { workspacePath } from "@/models/Workspace";

const cloneMemorySchema = z.object({
  destination: z.string().trim().min(1, "A destination is required"),
});

type CloneMemoryFormData = z.infer<typeof cloneMemorySchema>;

interface CloneMemoryDialogProps {
  memoryId: string;
  open: boolean;
  onClose: () => void;
}

// One per project, keeping the workspace slug and project token the clone's URL needs.
const cloneTargets = (destinations: CloneDestination[]) =>
  destinations.flatMap((d) =>
    d.projects
      .map((p) => ({ value: p.id, label: `${d.workspace.name} / ${p.name}`, slug: d.workspace.slug, token: projectToken(p) }))
      .sort((a, b) => a.label.localeCompare(b.label)),
  );

// Flattened, not option-grouped: the shared FormSelect has no group support, so each option's label carries
// its workspace prefix ("Engineering / Backend"). A memory always lands in a project.
export const CloneMemoryDialog = ({ memoryId, open, onClose }: CloneMemoryDialogProps) => {
  const navigate = useNavigate();
  const { data: destinations, error, isPending } = useFetchCloneDestinations();
  const clone = useCloneMemory();
  const form = useForm<CloneMemoryFormData>({
    resolver: zodResolver(cloneMemorySchema),
    defaultValues: { destination: "" },
  });

  const targets = cloneTargets(destinations ?? []);

  const onSubmit = async (data: CloneMemoryFormData) => {
    const target = targets.find((t) => t.value === data.destination);
    if (!target) return;
    try {
      const cloned = await clone.mutateAsync({ id: memoryId, projectId: target.value });
      onClose();
      // The copy may land in another workspace, so its URL takes that workspace's slug.
      navigate(workspacePath(target.slug, memoryPath(target.token, cloned.id)));
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
              options={targets}
              placeholder="Select a project"
            />
            <DialogFooter>
              <Button variant="outline" type="button" onClick={onClose}>
                Cancel
              </Button>
              <Button type="submit" loading={clone.isPending}>
                Clone
              </Button>
            </DialogFooter>
          </form>
        )}
      </DialogContent>
    </Dialog>
  );
};
