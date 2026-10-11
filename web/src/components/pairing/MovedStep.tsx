import { CommandSnippet } from "@/components/pairing/CommandSnippet";
import { cloudflaredRemovalCommands, MOVE_GUIDE } from "@/utils/MoveCommands";

interface MovedStepProps {
  name: string;
  // The computer was paired through a tunnel, so cloudflared is still installed on it.
  hadTunnel: boolean;
}

// The move is done: what changed, and the one command that takes cloudflared off a computer whose tunnel is gone.
export const MovedStep = ({ name, hadTunnel }: MovedStepProps) => (
  <div className="space-y-3">
    <p className="text-sm">
      {name} is paired through the Nexul app now, with its project links, setup and history as they were.
      {hadTunnel && " Its Cloudflare tunnel, DNS record and Access app are deleted."}
    </p>
    {hadTunnel && (
      <>
        <p className="text-sm text-muted-foreground">
          cloudflared still runs on the computer, connecting to a tunnel that is gone. Run this there to remove its service:
        </p>
        <CommandSnippet commands={cloudflaredRemovalCommands} label="Removal command" guide={MOVE_GUIDE} />
      </>
    )}
  </div>
);
