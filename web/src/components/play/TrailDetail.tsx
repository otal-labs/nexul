import { ErrorDisplay } from "@/components/ErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { TrailDetailBody } from "@/components/play/TrailDetailBody";
import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { useFetchTrail } from "@/hooks/TrailHooks";

interface TrailDetailProps {
  trailId: string | null;
  onClose: () => void;
}

// A large centred dialog: the run facts up top, the transcript taking the rest of the height and scrolling inside.
export const TrailDetail = ({ trailId, onClose }: TrailDetailProps) => {
  const { data: trail, error, isPending } = useFetchTrail(trailId);

  return (
    <Dialog open={trailId !== null} onOpenChange={(next) => !next && onClose()}>
      <DialogContent className="flex h-[85vh] max-h-[85vh] flex-col gap-0 p-0 sm:max-w-4xl">
        <DialogHeader className="px-6 pt-6 pb-4">
          <DialogTitle>{trail?.play_label ?? "Trail"}</DialogTitle>
          <DialogDescription>The choices made and every step the Agent took.</DialogDescription>
        </DialogHeader>
        {trailId !== null && isPending && <LoadingDisplay label="Loading trail…" />}
        {error && <ErrorDisplay error={error} title="Failed to load the trail" />}
        {trail && <TrailDetailBody trail={trail} />}
      </DialogContent>
    </Dialog>
  );
};
