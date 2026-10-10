import { zodResolver } from "@hookform/resolvers/zod";
import { useState } from "react";
import { useForm } from "react-hook-form";

import { PlayRunDialog } from "@/components/play/PlayRunDialog";
import { FormTextarea } from "@/components/ticket/FormTextarea";
import { Button } from "@/components/ui/button";
import { useFetchWorkspacePlays } from "@/hooks/PlayHooks";
import { useContinueTrail } from "@/hooks/TrailHooks";
import { isNeedsLocationRefusal } from "@/models/Pairing";
import { ContinueTrailFormSchema, type ContinueTrailFormData, type Trail } from "@/models/Trail";

interface TrailContinueFormProps {
  trail: Trail;
}

// Carries an ended run on in its own harness thread with only what is typed here; the run dialog asks where if that needs one.
export const TrailContinueForm = ({ trail }: TrailContinueFormProps) => {
  const { mutate, isPending } = useContinueTrail();
  const form = useForm<ContinueTrailFormData>({ defaultValues: { message: "" }, resolver: zodResolver(ContinueTrailFormSchema) });
  // The instructions the play starts again with once the person picks where; null while nothing asks.
  const [again, setAgain] = useState<string | null>(null);
  const { data: plays } = useFetchWorkspacePlays(again === null ? "" : trail.workspace_id);
  const play = plays?.find((p) => p.id === trail.play_id);

  const send = ({ message }: ContinueTrailFormData) =>
    mutate(
      { trailId: trail.id, message },
      {
        onSuccess: () => form.reset(),
        onError: (error) => {
          if (!isNeedsLocationRefusal(error)) return;
          setAgain([trail.custom_instructions, message].filter((part) => part !== "").join("\n\n"));
        },
      },
    );

  return (
    <>
      <form onSubmit={form.handleSubmit(send)} className="space-y-2 border-t border-border px-6 py-4">
        <FormTextarea
          control={form.control}
          name="message"
          label="Continue this run"
          placeholder="What should the agent do next? The play's instructions aren't sent again."
          rows={2}
          className="resize-none bg-background"
        />
        <div className="flex justify-end">
          <Button type="submit" size="sm" disabled={isPending}>
            {isPending ? "Sending…" : "Send"}
          </Button>
        </div>
      </form>
      {again !== null && play && (
        <PlayRunDialog
          play={play}
          projectId={trail.project_id}
          targetType={trail.target_type}
          targetId={trail.target_id}
          instructions={again}
          askWhere
          open
          onClose={() => setAgain(null)}
        />
      )}
    </>
  );
};
