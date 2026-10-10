import { PairCommands } from "@/components/pairing/PairCommands";
import { PairT3CodeForm } from "@/components/pairing/PairT3CodeForm";
import type { Computer } from "@/models/Pairing";

interface PairT3CodeStepProps {
  computer: Computer;
  onPaired: (computer: Computer) => void;
}

// Step two: trade the one-time pairing token for a session over the verified hostname.
export const PairT3CodeStep = ({ computer, onPaired }: PairT3CodeStepProps) => (
  <div className="max-w-xl space-y-5">
    <div className="space-y-3">
      <p className="text-sm text-muted-foreground">Nexul pairs with {computer.name} over its tunnel.</p>
      <PairCommands viaTunnel />
    </div>
    <PairT3CodeForm computer={computer} onPaired={onPaired} />
  </div>
);
