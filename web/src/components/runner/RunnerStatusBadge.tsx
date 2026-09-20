import { Wifi, WifiOff } from "lucide-react";

import { NoFillBadge } from "@/components/ui/badge";

interface RunnerStatusBadgeProps {
  connected: boolean;
}

// Disconnected stays neutral --muted-foreground, not --destructive: going offline isn't itself an error.
export const RunnerStatusBadge = ({ connected }: RunnerStatusBadgeProps) => (
  <NoFillBadge
    icon={connected ? Wifi : WifiOff}
    color={connected ? "text-success" : "text-muted-foreground"}
  >
    {connected ? "online" : "offline"}
  </NoFillBadge>
);
