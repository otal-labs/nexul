import { AddComputerStep } from "@/components/pairing/AddComputerStep";
import { ComputerChecks } from "@/components/pairing/ComputerChecks";
import { MovedStep } from "@/components/pairing/MovedStep";
import { PAIR_DIALOG_FRAME } from "@/components/pairing/PairDialogFrame";
import { ErrorDisplay } from "@/components/ErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { Button } from "@/components/ui/button";
import { Dialog, DialogClose, DialogContent, DialogDescription, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { usePairNow, type useEnrollComputer } from "@/hooks/ComputerHooks";
import { pairedThroughRunner } from "@/models/ComputerChecks";
import type { Computer } from "@/models/Pairing";

export type MoveFrom = "tunnel" | "url";

const leaving = (from: MoveFrom | undefined) => (from === "tunnel" ? "its Cloudflare tunnel is removed" : "Nexul stops reaching it at its address");

const lead = (name: string, from: MoveFrom | undefined, moved: boolean) => {
  if (moved) return `Nexul reaches T3 Code on ${name} through the Nexul app from now on, and keeps it paired.`;
  return `One command installs the Nexul app on ${name}. Once Nexul pairs T3 Code through it, the computer keeps its project links, setup and history, and ${leaving(from)}. Until then it keeps working as it is.`;
};

interface MoveComputerDialogProps {
  computer: Computer;
  // Where the computer is moving from; the dialog is open while it is set.
  from: MoveFrom | undefined;
  // The command the row made as it opened the dialog; none when the app is already installed there.
  enroll: ReturnType<typeof useEnrollComputer>;
  onClose: () => void;
}

// Move a computer paired through a tunnel or by URL onto the Nexul app: the command and its checks, then what is left to remove.
export const MoveComputerDialog = ({ computer, from, enroll, onClose }: MoveComputerDialogProps) => {
  const pairNow = usePairNow(computer);
  const moved = pairedThroughRunner(computer) && !computer.tunnel;
  const retry = !moved && computer.runner?.connected === true && !!computer.pair_error;

  return (
    <Dialog open={from !== undefined} onOpenChange={(open) => !open && onClose()}>
      <DialogContent className={PAIR_DIALOG_FRAME} onInteractOutside={(e) => e.preventDefault()}>
        <DialogHeader className="gap-3 border-b border-border px-4 pt-5 pb-4 text-left sm:px-6">
          <DialogTitle className="pr-8">{moved ? `${computer.name} moved to the Nexul app` : `Move ${computer.name} to the Nexul app`}</DialogTitle>
          <DialogDescription>{lead(computer.name, from, moved)}</DialogDescription>
        </DialogHeader>
        <div className="min-h-0 flex-1 overflow-y-auto px-4 py-5 sm:px-6">
          {enroll.isPending && <LoadingDisplay label="Making the command" />}
          {enroll.error && <ErrorDisplay error={enroll.error} title="Couldn't make the command." />}
          {!moved && enroll.data && <AddComputerStep enrollment={enroll.data} computer={computer} />}
          {!moved && enroll.isIdle && <ComputerChecks computer={computer} />}
          {moved && <MovedStep name={computer.name} hadTunnel={from === "tunnel"} />}
        </div>
        <div className="flex justify-end gap-2 border-t border-border px-4 py-3 sm:px-6">
          <DialogClose asChild>
            <Button type="button" variant="outline">
              {moved ? "Close" : "Cancel"}
            </Button>
          </DialogClose>
          {enroll.error && (
            <Button type="button" onClick={() => enroll.mutate(computer.id)}>
              Try again
            </Button>
          )}
          {retry && (
            <Button type="button" loading={pairNow.isPending} onClick={() => pairNow.mutate()}>
              Pair now
            </Button>
          )}
        </div>
      </DialogContent>
    </Dialog>
  );
};
