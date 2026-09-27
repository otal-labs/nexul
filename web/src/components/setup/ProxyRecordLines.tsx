import type { PublicAddress } from "@/models/Setup";

interface ProxyRecordLinesProps {
  domain: string;
  address: PublicAddress;
}

// The exact records to create at the DNS provider, one mono line each, in the provider's own column order.
export const ProxyRecordLines = ({ domain, address }: ProxyRecordLinesProps) => (
  <div className="space-y-1.5">
    <p className="text-sm text-muted-foreground">Create this at your DNS provider:</p>
    <ul className="divide-y divide-border rounded-md border border-border font-mono text-xs" aria-label="DNS records">
      <li className="flex flex-wrap gap-x-3 px-3 py-2">
        <span className="w-10 shrink-0">A</span>
        <span className="min-w-0 break-all">{domain || "your domain"}</span>
        <span className="text-muted-foreground">{address.ipv4}</span>
      </li>
      {address.ipv6 && (
        <li className="flex flex-wrap gap-x-3 px-3 py-2">
          <span className="w-10 shrink-0">AAAA</span>
          <span className="min-w-0 break-all">{domain || "your domain"}</span>
          <span className="break-all text-muted-foreground">{address.ipv6}</span>
        </li>
      )}
    </ul>
  </div>
);
