import { useState, type ReactNode } from "react";

import { AddComputerStep } from "@/components/pairing/AddComputerStep";
import { PAIR_DIALOG_FRAME, SETUP_DIALOG_FRAME, SETUP_LEAD } from "@/components/pairing/PairDialogFrame";
import { SetupDoneButton } from "@/components/pairing/SetupDoneButton";
import { SetupStep } from "@/components/pairing/SetupStep";
import { Button } from "@/components/ui/button";
import { Dialog, DialogClose, DialogContent, DialogDescription, DialogHeader, DialogTitle, DialogTrigger } from "@/components/ui/dialog";
import { ErrorDisplay } from "@/components/ErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { useEnrollComputer } from "@/hooks/ComputerHooks";
import { useListComputers } from "@/hooks/PairingHooks";
import { useSetupDraftStore } from "@/stores/setupDraftStore";
import { stillPairing } from "@/models/Pairing";
import { cn } from "@/lib/utils";

type Step = "add" | "setup";

const ADD_LEAD = "One command installs the Nexul app on the computer. Nexul reaches T3 Code through it, so nothing on the computer opens to the internet.";

const dialogTitle = (step: Step, name: string | undefined, again: AddComputerDialogProps["again"]) => {
  if (step === "setup") return `Set up ${name || "this computer"}`;
  if (again?.name) return `Add ${again.name} again`;
  if (again) return "Add this computer again";
  return "Add a computer";
};

interface AddComputerDialogProps {
  // Absent while a row has nothing to add, so the dialog stays open as the row it was opened from changes.
  trigger?: ReactNode;
  // A computer whose app is gone: the dialog makes a fresh command for that row instead of adding another.
  again?: { id: string; name: string } | undefined;
}

// Add a computer: a one-time command, three live checks as the computer connects and pairs, then Set up.
export const AddComputerDialog = ({ trigger, again }: AddComputerDialogProps) => {
  const [open, setOpen] = useState(false);
  const [step, setStep] = useState<Step>("add");
  const enroll = useEnrollComputer();
  const { data: computers } = useListComputers();
  const added = enroll.data?.computer;
  const computer = computers?.find((c) => c.id === added?.id) ?? added;
  const paired = !!computer && !stillPairing(computer);

  // Opening mints the command; every way out but Done drops the Set up step's unsaved edits.
  const onOpenChange = (next: boolean) => {
    setOpen(next);
    if (next) {
      enroll.mutate(again?.id);
      return;
    }
    if (computer) useSetupDraftStore.getState().discard(computer.id);
    setStep("add");
    enroll.reset();
  };


  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      {trigger && <DialogTrigger asChild>{trigger}</DialogTrigger>}
      <DialogContent
        className={cn(PAIR_DIALOG_FRAME, step === "setup" && SETUP_DIALOG_FRAME)}
        // A stray click beside the command or a long setup must not throw the dialog away; Cancel and Esc close it on purpose.
        onInteractOutside={(e) => e.preventDefault()}
      >
        <DialogHeader className="gap-3 border-b border-border px-4 pt-5 pb-4 text-left sm:px-6">
          <DialogTitle className="pr-8">{dialogTitle(step, computer?.name || again?.name, again)}</DialogTitle>
          <DialogDescription>{step === "setup" ? SETUP_LEAD : ADD_LEAD}</DialogDescription>
        </DialogHeader>
        <div className="min-h-0 flex-1 overflow-y-auto px-4 py-5 sm:px-6">
          {enroll.isPending && <LoadingDisplay label="Making the command" />}
          {enroll.error && <ErrorDisplay error={enroll.error} title="Couldn't make the command." />}
          {enroll.data && computer && step === "add" && <AddComputerStep enrollment={enroll.data} computer={computer} />}
          {computer && step === "setup" && <SetupStep computer={computer} />}
        </div>
        <div className="flex justify-end gap-2 border-t border-border px-4 py-3 sm:px-6">
          <DialogClose asChild>
            <Button type="button" variant="outline">
              Cancel
            </Button>
          </DialogClose>
          {enroll.error && (
            <Button type="button" onClick={() => enroll.mutate(again?.id)}>
              Try again
            </Button>
          )}
          {enroll.data && step === "add" && (
            <Button type="button" disabled={!paired} onClick={() => setStep("setup")}>
              Set up
            </Button>
          )}
          {computer && step === "setup" && <SetupDoneButton computerId={computer.id} onDone={() => onOpenChange(false)} />}
        </div>
      </DialogContent>
    </Dialog>
  );
};
