import { useState, type ReactNode } from "react";

import { ConnectStep } from "@/components/pairing/ConnectStep";
import { PAIR_DIALOG_FRAME, SETUP_DIALOG_FRAME, SETUP_LEAD } from "@/components/pairing/PairDialogFrame";
import { PairingStepTabs } from "@/components/pairing/PairingStepTabs";
import { PairT3CodeStep } from "@/components/pairing/PairT3CodeStep";
import { SetupDoneButton } from "@/components/pairing/SetupDoneButton";
import { SetupStep } from "@/components/pairing/SetupStep";
import { Button } from "@/components/ui/button";
import { Dialog, DialogClose, DialogContent, DialogDescription, DialogHeader, DialogTitle, DialogTrigger } from "@/components/ui/dialog";
import { Tabs, TabsContent } from "@/components/ui/tabs";
import { useFetchTunnelStatus } from "@/hooks/PairingHooks";
import { useSetupDraftStore } from "@/stores/setupDraftStore";
import { stillPairing, tunnelConnected, type Computer, type PairingStep } from "@/models/Pairing";
import { cn } from "@/lib/utils";

interface NextButtonProps {
  computerId: string;
  onNext: () => void;
}

const NextButton = ({ computerId, onNext }: NextButtonProps) => {
  const { data: status } = useFetchTunnelStatus(computerId);
  return (
    <Button type="button" onClick={onNext} disabled={!status || !tunnelConnected(status)}>
      Next
    </Button>
  );
};

// The furthest step the tabs may open: Set up only once paired, Pair T3 Code only once Next led there.
const reachableStep = (step: PairingStep, paired: Computer | undefined): PairingStep => {
  if (paired) return "setup";
  if (step === "connect") return "connect";
  return "pair";
};

const PAIR_LEAD = "The computer keeps a tunnel open to this instance's Cloudflare, so Nexul can reach T3 Code on it from anywhere.";

interface PairComputerDialogProps {
  // Absent for a row that opens it from its own button or menu.
  trigger?: ReactNode;
  // A paired computer opens straight at Set up; one still pairing through its tunnel resumes at Connect.
  existing: Computer;
  // onClosed lets a link straight to a computer's Set up step forget it.
  onClosed?: (() => void) | undefined;
  open?: boolean | undefined;
  onOpenChange?: ((open: boolean) => void) | undefined;
}

// An existing computer's dialog: a tunnel computer finishes pairing over its tunnel, and every paired computer sets up here.
export const PairComputerDialog = ({ trigger, existing, onClosed, open: controlled, onOpenChange: notify }: PairComputerDialogProps) => {
  const setupFor = stillPairing(existing) ? undefined : existing;
  const first: PairingStep = setupFor ? "setup" : "connect";
  const [uncontrolled, setOpen] = useState(false);
  const open = controlled ?? uncontrolled;
  const [step, setStep] = useState<PairingStep>(first);
  const [paired, setPaired] = useState<Computer | undefined>(setupFor);

  // Every way out but Done is Cancel: Esc, the corner close, and the Cancel button drop the Set up step's unsaved edits.
  const onOpenChange = (next: boolean) => {
    setOpen(next);
    notify?.(next);
    if (paired) useSetupDraftStore.getState().discard(paired.id);
    if (next) return;
    onClosed?.();
    setStep(first);
    setPaired(setupFor);
  };

  const onPaired = (c: Computer) => {
    setPaired(c);
    setStep("setup");
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      {trigger && <DialogTrigger asChild>{trigger}</DialogTrigger>}
      <DialogContent
        className={cn(PAIR_DIALOG_FRAME, step === "setup" && SETUP_DIALOG_FRAME)}
        // A stray click beside a long setup must not throw the step's choices away; Cancel and Esc close it on purpose.
        onInteractOutside={(e) => e.preventDefault()}
      >
        <Tabs value={step} onValueChange={(v) => setStep(v as PairingStep)} className="flex min-h-0 flex-1 flex-col gap-0">
          <DialogHeader className="gap-3 border-b border-border px-4 pt-5 pb-4 text-left sm:px-6">
            <DialogTitle className="pr-8">{setupFor ? `Set up ${existing.name}` : `Pair ${existing.name}`}</DialogTitle>
            <DialogDescription>{setupFor ? SETUP_LEAD : PAIR_LEAD}</DialogDescription>
            {first === "connect" && <PairingStepTabs step={step} reachable={reachableStep(step, paired)} />}
          </DialogHeader>
          <div className="min-h-0 flex-1 overflow-y-auto px-4 py-5 sm:px-6">
            <TabsContent value="connect">
              <ConnectStep computer={existing} />
            </TabsContent>
            <TabsContent value="pair">
              <PairT3CodeStep computer={existing} onPaired={onPaired} />
            </TabsContent>
            <TabsContent value="setup" tabIndex={-1}>
              {paired && <SetupStep computer={paired} />}
            </TabsContent>
          </div>
        </Tabs>
        <div className="flex justify-end gap-2 border-t border-border px-4 py-3 sm:px-6">
          <DialogClose asChild>
            <Button type="button" variant="outline">
              Cancel
            </Button>
          </DialogClose>
          {step === "connect" && <NextButton computerId={existing.id} onNext={() => setStep("pair")} />}
          {step === "setup" && paired && <SetupDoneButton computerId={paired.id} onDone={() => onOpenChange(false)} />}
        </div>
      </DialogContent>
    </Dialog>
  );
};
