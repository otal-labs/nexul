import { useState, type ReactNode } from "react";

import { ConnectStep } from "@/components/pairing/ConnectStep";
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

// Centred dialog from 640px up, a full-screen sheet below it.
const FRAME =
  "flex max-h-[min(90dvh,52rem)] flex-col gap-0 p-0 sm:max-w-[min(48rem,calc(100%-2rem))] max-sm:inset-0 max-sm:top-0 max-sm:left-0 max-sm:h-dvh max-sm:max-h-none max-sm:max-w-none max-sm:translate-x-0 max-sm:translate-y-0 max-sm:rounded-none max-sm:border-0";

// Set up is two panes from md up: wider and taller, at a fixed height so the panes hold still while the transcript streams.
const SETUP_FRAME = "md:h-[min(90dvh,56rem)] md:max-h-[min(90dvh,56rem)] md:max-w-[min(80rem,calc(100%-4rem))]";

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

const dialogTitle = (existing: Computer | undefined, setupFor: Computer | undefined) => {
  if (setupFor) return `Set up ${setupFor.name}`;
  if (existing) return `Pair ${existing.name}`;
  return "Pair a computer";
};

interface PairComputerDialogProps {
  // Absent for a row that only opens the dialog while its computer is still pairing.
  trigger?: ReactNode;
  // A paired computer opens straight at Set up; one still pairing resumes at Connect with its tunnel.
  existing?: Computer | undefined;
  // Opens on mount, for a link straight to a computer's Set up step; onClosed lets that link's URL forget it.
  defaultOpen?: boolean | undefined;
  onClosed?: (() => void) | undefined;
}

// Pair a computer: connect its tunnel, pair T3 Code over it, then set it up.
export const PairComputerDialog = ({ trigger, existing, defaultOpen = false, onClosed }: PairComputerDialogProps) => {
  const setupFor = existing && !stillPairing(existing) ? existing : undefined;
  const first: PairingStep = setupFor ? "setup" : "connect";
  const [open, setOpen] = useState(defaultOpen);
  const [step, setStep] = useState<PairingStep>(first);
  const [computer, setComputer] = useState<Computer | undefined>(existing);
  const [paired, setPaired] = useState<Computer | undefined>(setupFor);

  // Every way out but Done is Cancel: Esc, the corner close, and the Cancel button drop the Set up step's unsaved edits.
  const onOpenChange = (next: boolean) => {
    setOpen(next);
    if (paired) useSetupDraftStore.getState().discard(paired.id);
    if (next) return;
    onClosed?.();
    setStep(first);
    setComputer(existing);
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
        className={cn(FRAME, step === "setup" && SETUP_FRAME)}
        // A stray click beside a long setup must not throw the step's choices away; Cancel and Esc close it on purpose.
        onInteractOutside={(e) => e.preventDefault()}
      >
        <Tabs value={step} onValueChange={(v) => setStep(v as PairingStep)} className="flex min-h-0 flex-1 flex-col gap-0">
          <DialogHeader className="gap-3 border-b border-border px-4 pt-5 pb-4 text-left sm:px-6">
            <DialogTitle className="pr-8">{dialogTitle(existing, setupFor)}</DialogTitle>
            <DialogDescription>{setupFor ? SETUP_LEAD : PAIR_LEAD}</DialogDescription>
            <PairingStepTabs step={step} reachable={reachableStep(step, paired)} earliest={first} />
          </DialogHeader>
          <div className={cn("min-h-0 flex-1 overflow-y-auto px-4 py-5 sm:px-6", step === "setup" && "md:overflow-hidden md:p-0")}>
            <TabsContent value="connect">
              <ConnectStep computer={computer} onCreated={setComputer} onPairByUrl={() => setStep("pair")} />
            </TabsContent>
            <TabsContent value="pair">
              <PairT3CodeStep computer={computer} onPaired={onPaired} />
            </TabsContent>
            <TabsContent value="setup" className="md:h-full">
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
          {step === "connect" && <NextButton computerId={computer?.id ?? ""} onNext={() => setStep("pair")} />}
          {step === "setup" && paired && <SetupDoneButton computerId={paired.id} onDone={() => onOpenChange(false)} />}
        </div>
      </DialogContent>
    </Dialog>
  );
};
