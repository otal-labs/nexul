import { LaptopIcon } from "lucide-react";
import { Link } from "react-router";

import { EmptyState } from "@/components/EmptyState";
import { Button } from "@/components/ui/button";
import { computersSearch } from "@/models/Agent";

// The move notice opens on what to do and where; each computer's own pane in Computers carries the command.
export const ComputerMoveDetail = () => {
  return (
    <EmptyState
      icon={LaptopIcon}
      title="Move your computers to the Nexul app"
      message="Pairing through a Cloudflare tunnel or by URL was retired, so agent work runs only on computers added with the Nexul app. In Computers, open each computer paired that way and press Move: one command installs the Nexul app, and Nexul pairs the computer through it, keeping its projects, setup and history."
      action={
        <Button asChild size="sm">
          <Link to={computersSearch()}>Open Computers</Link>
        </Button>
      }
    />
  );
};
