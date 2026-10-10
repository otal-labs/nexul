import { PairCommands } from "@/components/pairing/PairCommands";
import { PairT3CodeForm } from "@/components/pairing/PairT3CodeForm";
import type { Computer } from "@/models/Pairing";

interface PairT3CodeStepProps {
  computer: Computer | undefined;
  onPaired: (computer: Computer) => void;
}

// Step two: trade the one-time pairing token for a session, over the verified hostname or, from Advanced, a URL the server reaches.
export const PairT3CodeStep = ({ computer, onPaired }: PairT3CodeStepProps) => {
  const lead = computer
    ? `Nexul pairs with ${computer.name} over its tunnel.`
    : "For a machine this server can already reach. Enter its T3 Code URL below.";
  return (
    <div className="max-w-xl space-y-5">
      <div className="space-y-3">
        <p className="text-sm text-muted-foreground">{lead}</p>
        <PairCommands />
      </div>
      <PairT3CodeForm key={computer?.id ?? "url"} computer={computer} onPaired={onPaired} />
    </div>
  );
};
