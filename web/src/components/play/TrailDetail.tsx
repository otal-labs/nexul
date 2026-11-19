import { ErrorDisplay } from "@/components/ErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { TrailDetailBody } from "@/components/play/TrailDetailBody";
import { TrailHeadline } from "@/components/play/TrailHeadline";
import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { useFetchTrail } from "@/hooks/TrailHooks";
import { QuestionDockContext } from "@/hooks/useQuestionDock";

interface TrailDetailProps {
  trailId: string | null;
  onClose: () => void;
}

// A large centred dialog: the run's state and facts up top, the transcript taking the rest of the height and scrolling inside.
export const TrailDetail = ({ trailId, onClose }: TrailDetailProps) => {
  const { data: trail, error, isPending } = useFetchTrail(trailId);

  return (
    <Dialog open={trailId !== null} onOpenChange={(next) => !next && onClose()}>
      <DialogContent className="flex h-[85vh] max-h-[85vh] flex-col gap-0 p-0 sm:max-w-4xl">
        <DialogHeader className="gap-2 px-6 pt-6 pb-5">
          <DialogTitle>{trail?.play_label ?? "Play run"}</DialogTitle>
          {trail && <TrailHeadline trail={trail} />}
          {!trail && <DialogDescription className="sr-only">The run's state, where it ran and every step the Agent took.</DialogDescription>}
        </DialogHeader>
        {trailId !== null && isPending && <LoadingDisplay label="Loading the run…" />}
        {error && <ErrorDisplay error={error} title="Couldn't load the run." />}
        {/* A run opened from a thread answers its question here, not in the thread's composer behind the dialog. */}
        {trail && (
          <QuestionDockContext value={null}>
            <TrailDetailBody trail={trail} />
          </QuestionDockContext>
        )}
      </DialogContent>
    </Dialog>
  );
};
