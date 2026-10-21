import { LaptopIcon } from "lucide-react";
import { Link } from "react-router";

import { EmptyState } from "@/components/EmptyState";
import { Button } from "@/components/ui/button";

// The move notice opens on what to do and where; each computer's own row in Computers carries the command.
export const ComputerMoveDetail = () => (
  <EmptyState
    icon={LaptopIcon}
    title="Move your computers to the Nexul app"
    message="Pairing through a Cloudflare tunnel or by URL was retired, so agent work runs only on computers added with the Nexul app. In Computers, press Move on each computer paired that way: one command installs the Nexul app, and Nexul pairs the computer through it, keeping its projects, setup and history."
    action={
      <Button asChild size="sm">
        <Link to="/settings/pairing">Open Computers</Link>
      </Button>
    }
  />
);
