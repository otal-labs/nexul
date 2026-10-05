import { zodResolver } from "@hookform/resolvers/zod";
import { useForm } from "react-hook-form";

import { FormTextarea } from "@/components/ticket/FormTextarea";
import { Button } from "@/components/ui/button";
import { useContinueTrail } from "@/hooks/TrailHooks";
import { ContinueTrailFormSchema, type ContinueTrailFormData } from "@/models/Trail";

interface TrailContinueFormProps {
  trailId: string;
}

// Carries an ended run on in its own harness thread with only what is typed here, instead of pressing the play again.
export const TrailContinueForm = ({ trailId }: TrailContinueFormProps) => {
  const { mutate, isPending } = useContinueTrail();
  const form = useForm<ContinueTrailFormData>({ defaultValues: { message: "" }, resolver: zodResolver(ContinueTrailFormSchema) });

  return (
    <form
      onSubmit={form.handleSubmit(({ message }) => mutate({ trailId, message }, { onSuccess: () => form.reset() }))}
      className="space-y-2 border-t border-border px-6 py-4"
    >
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
  );
};
