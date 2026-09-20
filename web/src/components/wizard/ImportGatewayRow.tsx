import { CheckCircle2 } from "lucide-react";

import { NoFillBadge } from "@/components/ui/badge";
import { ImportContainerRow } from "@/components/wizard/ImportContainerRow";
import type { DiscoveredContainer } from "@/models/Machine";

interface ImportGatewayRowProps {
  container: DiscoveredContainer;
  checked: boolean;
  onToggle: () => void;
}

// A recognised gateway row plus what its tunnel already serves, so an existing tunnel's hostnames are visible
// before anything is adopted; a describe failure is shown on the row, never as a blocking error.
export const ImportGatewayRow = ({ container, checked, onToggle }: ImportGatewayRowProps) => {
  const { tunnel, tunnel_error: tunnelError } = container;
  const idle = tunnel && tunnel.routes.length === 0 && tunnel.records.length === 0;

  return (
    <div className="space-y-1.5">
      <ImportContainerRow container={container} checked={checked} onToggle={onToggle} />
      {tunnelError && <p className="pl-7 text-xs text-destructive">Could not read the tunnel: {tunnelError}</p>}
      {tunnel && (
        <div className="space-y-1 pl-7 text-xs">
          <div className="flex flex-wrap items-center gap-2">
            <span>
              Tunnel <span className="font-medium">{tunnel.name}</span>
              <span className="text-muted-foreground"> · {tunnel.status}</span>
            </span>
            {tunnel.tracked && (
              <NoFillBadge icon={CheckCircle2} color="text-success">
                Tracked by this instance
              </NoFillBadge>
            )}
          </div>
          {idle && <p className="text-muted-foreground">No hostnames routed through it yet.</p>}
          {tunnel.routes.length > 0 && (
            <ul aria-label={`Hostnames routed through ${tunnel.name}`} className="space-y-0.5">
              {tunnel.routes.map((route) => (
                <li key={route.hostname} className="font-mono text-muted-foreground">
                  {route.hostname} → {route.service}
                </li>
              ))}
            </ul>
          )}
          {tunnel.records.length > 0 && (
            <ul aria-label={`DNS records pointing at ${tunnel.name}`} className="space-y-0.5">
              {tunnel.records.map((record) => (
                <li key={record.name} className="font-mono text-muted-foreground">
                  {record.name} CNAME {record.content}
                </li>
              ))}
            </ul>
          )}
        </div>
      )}
    </div>
  );
};
