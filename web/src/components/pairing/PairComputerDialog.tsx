import { useState, type ReactNode } from "react";

import { ConnectStep } from "@/components/pairing/ConnectStep";
import { PairingStepTabs } from "@/components/pairing/PairingStepTabs";
import { PairT3CodeStep } from "@/components/pairing/PairT3CodeStep";
import { SetupStep } from "@/components/pairing/SetupStep";
import { Button } from "@/components/ui/button";
import { Dialog, DialogClose, DialogContent, DialogDescription, DialogHeader, DialogTitle, DialogTrigger } from "@/components/ui/dialog";
import { Tabs, TabsContent } from "@/components/ui/tabs";
import { useFetchTunnelStatus } from "@/hooks/PairingHooks";
import { tunnelConnected, type Computer, type PairingStep } from "@/models/Pairing";

// Centred dialog from 640px up, a full-screen sheet below it.
const FRAME =
  "flex max-h-[min(90dvh,52rem)] flex-col gap-0 p-0 sm:max-w-[min(48rem,calc(100%-2rem))] max-sm:inset-0 max-sm:top-0 max-sm:left-0 max-sm:h-dvh max-sm:max-h-none max-sm:max-w-none max-sm:translate-x-0 max-sm:translate-y-0 max-sm:rounded-none max-sm:border-0";

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

// The furthest step the tabs may open: Set up only once paired, Pair T3 Code only once Next or Pair by URL led there.
const reachableStep = (step: PairingStep, paired: Computer | undefined): PairingStep => {
  if (paired) return "setup";
  if (step === "connect") return "connect";
  return "pair";
};

const PAIR_LEAD = "The computer keeps a tunnel open to this instance's Cloudflare, so Nexul can reach T3 Code on it from anywhere.";
const SETUP_LEAD = "Agent work runs on this computer once each provider on it is set up and confirmed.";

interface PairComputerDialogProps {
  trigger: ReactNode;
  // A paired computer's row opens the dialog straight at Set up for it, with the earlier steps done.
  setupFor?: Computer | undefined;
  // Opens on mount, for a link straight to a computer's Set up step; onClosed lets that link's URL forget it.
  defaultOpen?: boolean | undefined;
  onClosed?: (() => void) | undefined;
}

// Pair a computer: connect its tunnel, pair T3 Code over it, then set it up.
export const PairComputerDialog = ({ trigger, setupFor, defaultOpen = false, onClosed }: PairComputerDialogProps) => {
  const first: PairingStep = setupFor ? "setup" : "connect";
  const [open, setOpen] = useState(defaultOpen);
  const [step, setStep] = useState<PairingStep>(first);
  const [computer, setComputer] = useState<Computer | undefined>(setupFor);
  const [paired, setPaired] = useState<Computer | undefined>(setupFor);

  const onOpenChange = (next: boolean) => {
    setOpen(next);
    if (next) return;
    onClosed?.();
    setStep(first);
    setComputer(setupFor);
    setPaired(setupFor);
  };

  const onPaired = (c: Computer) => {
    setPaired(c);
    setStep("setup");
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogTrigger asChild>{trigger}</DialogTrigger>
      <DialogContent className={FRAME}>
        <Tabs value={step} onValueChange={(v) => setStep(v as PairingStep)} className="flex min-h-0 flex-1 flex-col gap-0">
          <DialogHeader className="gap-3 border-b border-border px-4 pt-5 pb-4 text-left sm:px-6">
            <DialogTitle className="pr-8">{setupFor ? `Set up ${setupFor.name}` : "Pair a computer"}</DialogTitle>
            <DialogDescription>{setupFor ? SETUP_LEAD : PAIR_LEAD}</DialogDescription>
            <PairingStepTabs step={step} reachable={reachableStep(step, paired)} earliest={first} />
          </DialogHeader>
          <div className="min-h-0 flex-1 overflow-y-auto px-4 py-5 sm:px-6">
            <TabsContent value="connect">
              <ConnectStep computer={computer} onCreated={setComputer} onPairByUrl={() => setStep("pair")} />
            </TabsContent>
            <TabsContent value="pair">
              <PairT3CodeStep computer={computer} onPaired={onPaired} />
            </TabsContent>
            <TabsContent value="setup">
              {paired && <SetupStep computer={paired} />}
            </TabsContent>
          </div>
        </Tabs>
        {step === "connect" && (
          <div className="flex justify-end gap-2 border-t border-border px-4 py-3 sm:px-6">
            <NextButton computerId={computer?.id ?? ""} onNext={() => setStep("pair")} />
          </div>
        )}
        {step === "setup" && (
          <div className="flex justify-end gap-2 border-t border-border px-4 py-3 sm:px-6">
            <DialogClose asChild>
              <Button type="button">Done</Button>
            </DialogClose>
          </div>
        )}
      </DialogContent>
    </Dialog>
  );
};
