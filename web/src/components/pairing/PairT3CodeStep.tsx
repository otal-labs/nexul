import { CommandBlock } from "@/components/pairing/CommandBlock";
import { PairT3CodeForm } from "@/components/pairing/PairT3CodeForm";
import type { Computer } from "@/models/Pairing";

interface PairT3CodeStepProps {
  computer: Computer | undefined;
  onPaired: (computer: Computer) => void;
}

// Step two: trade the `t3 pair` token for a session, over the verified hostname or, from Advanced, a URL the server reaches.
export const PairT3CodeStep = ({ computer, onPaired }: PairT3CodeStepProps) => {
  const lead = computer
    ? `Run this on ${computer.name}, then paste the one-time token it prints. Nexul pairs over the computer's tunnel.`
    : "For a machine this server can already reach, such as a VPS or a computer on the same network. Run this on it, then enter its T3 Code URL and the one-time token it prints.";
  return (
    <div className="max-w-xl space-y-5">
      <div className="space-y-3">
        <p className="text-sm text-muted-foreground">{lead}</p>
        <CommandBlock shell="Terminal" lines={["t3 pair"]} />
      </div>
      <PairT3CodeForm key={computer?.id ?? "url"} computer={computer} onPaired={onPaired} />
    </div>
  );
};
