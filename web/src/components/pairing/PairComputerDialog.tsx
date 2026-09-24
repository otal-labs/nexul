import { PlusIcon } from "lucide-react";
import { useState } from "react";

import { ConnectStep } from "@/components/pairing/ConnectStep";
import { PairingStepTabs } from "@/components/pairing/PairingStepTabs";
import { PairT3CodeStep } from "@/components/pairing/PairT3CodeStep";
import { EmptyRow } from "@/components/EmptyRow";
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

// Pair a computer: connect its tunnel, pair T3 Code over it, then set it up. Set up's content is still to come.
export const PairComputerDialog = () => {
  const [open, setOpen] = useState(false);
  const [step, setStep] = useState<PairingStep>("connect");
  const [computer, setComputer] = useState<Computer>();
  const [paired, setPaired] = useState<Computer>();

  const onOpenChange = (next: boolean) => {
    setOpen(next);
    if (next) return;
    setStep("connect");
    setComputer(undefined);
    setPaired(undefined);
  };

  const onPaired = (c: Computer) => {
    setPaired(c);
    setStep("setup");
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogTrigger asChild>
        <Button type="button">
          <PlusIcon className="size-4" aria-hidden />
          Pair a computer
        </Button>
      </DialogTrigger>
      <DialogContent className={FRAME}>
        <Tabs value={step} onValueChange={(v) => setStep(v as PairingStep)} className="flex min-h-0 flex-1 flex-col gap-0">
          <DialogHeader className="gap-3 border-b border-border px-4 pt-5 pb-4 text-left sm:px-6">
            <DialogTitle className="pr-8">Pair a computer</DialogTitle>
            <DialogDescription>
              The computer keeps a tunnel open to this instance's Cloudflare, so Nexul can reach T3 Code on it from anywhere.
            </DialogDescription>
            <PairingStepTabs step={step} reachable={reachableStep(step, paired)} />
          </DialogHeader>
          <div className="min-h-0 flex-1 overflow-y-auto px-4 py-5 sm:px-6">
            <TabsContent value="connect">
              <ConnectStep computer={computer} onCreated={setComputer} onPairByUrl={() => setStep("pair")} />
            </TabsContent>
            <TabsContent value="pair">
              <PairT3CodeStep computer={computer} onPaired={onPaired} />
            </TabsContent>
            <TabsContent value="setup">
              <EmptyRow>{paired?.name} is paired. Setting up its providers and skills comes next.</EmptyRow>
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
