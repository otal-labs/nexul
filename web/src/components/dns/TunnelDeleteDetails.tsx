import type { Tunnel } from "@/models/DNS";

interface TunnelDeleteDetailsProps {
  tunnel: Tunnel;
}

// What the confirm says goes with the tunnel: itself at Cloudflare, and the route and CNAME it created.
export const TunnelDeleteDetails = ({ tunnel }: TunnelDeleteDetailsProps) => (
  <ul className="space-y-1.5 border-y border-border py-3 text-sm">
    <li>
      The tunnel <span className="font-mono">{tunnel.name}</span> at Cloudflare
    </li>
    {tunnel.hostname && (
      <li>
        Its route <span className="font-mono break-all">{tunnel.hostname}</span> and the DNS record pointing at the tunnel
      </li>
    )}
  </ul>
);
