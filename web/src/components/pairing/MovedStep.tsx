import { CommandSnippet } from "@/components/pairing/CommandSnippet";
import { cloudflaredRemovalCommands, MOVE_GUIDE } from "@/utils/MoveCommands";

interface MovedStepProps {
  name: string;
}

// The move is done: what changed, and the one command that takes cloudflared off a computer that had a tunnel.
export const MovedStep = ({ name }: MovedStepProps) => (
  <div className="space-y-3">
    <p className="text-sm">{name} is paired through the Nexul app now, with its project links, setup and history as they were.</p>
    <p className="text-sm text-muted-foreground">
      If it was paired through a tunnel, cloudflared still runs on it, connecting to a tunnel that is gone. Run this there to remove its service:
    </p>
    <CommandSnippet commands={cloudflaredRemovalCommands} label="Removal command" guide={MOVE_GUIDE} />
  </div>
);
