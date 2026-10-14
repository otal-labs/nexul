import { CloudIcon, Trash2 } from "lucide-react";

import { TunnelDeleteDetails } from "@/components/dns/TunnelDeleteDetails";
import { SettingsStatus } from "@/components/settings/SettingsStatus";
import { ActionButton } from "@/components/action/ActionButton";
import { useDeleteTunnel } from "@/hooks/DnsHooks";
import { useConfirmationDialog } from "@/hooks/useConfirmationDialog";
import type { Tunnel } from "@/models/DNS";

interface TunnelRowProps {
  tunnel: Tunnel;
}

// Delete is refused by the server while a gateway or hostname uses the tunnel; the mutation's toast names them.
export const TunnelRow = ({ tunnel }: TunnelRowProps) => {
  const deleteTunnel = useDeleteTunnel();
  const { open: confirmDelete } = useConfirmationDialog();

  const onDelete = async () => {
    const ok = await confirmDelete({
      title: `Delete tunnel ${tunnel.name}?`,
      message: "This can't be undone.",
      details: <TunnelDeleteDetails tunnel={tunnel} />,
      confirmLabel: "Delete tunnel",
    });
    if (ok) deleteTunnel.mutate(tunnel.id);
  };

  return (
    <li className="flex items-start gap-3 px-3 py-3.5 transition-colors duration-150 ease-standard hover:bg-accent/40">
      <span className="flex size-9 shrink-0 items-center justify-center rounded-md border border-border bg-muted/60">
        <CloudIcon className="size-4 text-muted-foreground" aria-hidden />
      </span>
      <div className="min-w-0 flex-1 space-y-1">
        <p className="font-mono text-sm break-all">{tunnel.name}</p>
        {tunnel.hostname && <p className="font-mono text-xs break-all text-foreground/80">{tunnel.hostname}</p>}
        {!tunnel.hostname && <SettingsStatus tone="muted">No hostname routed</SettingsStatus>}
      </div>
      <ActionButton
        permission="dns:delete"
        verb="delete tunnels"
        type="button"
        variant="ghost"
        size="icon"
        icon={Trash2}
        aria-label={`Delete tunnel ${tunnel.name}`}
        title={`Delete tunnel ${tunnel.name}`}
        className="size-8"
        liveClassName="hover:text-destructive"
        loading={deleteTunnel.isPending}
        onClick={() => void onDelete()}
      />
    </li>
  );
};
