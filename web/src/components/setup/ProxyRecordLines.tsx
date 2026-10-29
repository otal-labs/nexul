import { microheaderClass } from "@/components/Microheader";
import type { PublicAddress } from "@/models/Setup";
import { cn } from "@/lib/utils";

interface ProxyRecordLinesProps {
  domain: string;
  address: PublicAddress;
}

// Type, name and value share one grid, so the columns read as a table rather than as gaps.
const RECORD_ROW = "grid grid-cols-[3rem_minmax(0,1fr)_auto] gap-x-3 px-3 py-2";

const RecordLine = ({ type, name, value }: { type: string; name: string; value: string }) => (
  <li className={RECORD_ROW}>
    <span>{type}</span>
    <span className="min-w-0 break-all">{name}</span>
    <span className="break-all text-muted-foreground">{value}</span>
  </li>
);

// The exact records to create at the DNS provider, in the provider's own column order.
export const ProxyRecordLines = ({ domain, address }: ProxyRecordLinesProps) => (
  <div className="space-y-1.5">
    <p className="text-sm text-muted-foreground">Create this at your DNS provider:</p>
    <ul className="divide-y divide-border rounded-md border border-border font-mono text-xs" aria-label="DNS records">
      <li aria-hidden className={cn(RECORD_ROW, microheaderClass, "py-1.5")}>
        <span>Type</span>
        <span>Name</span>
        <span>Value</span>
      </li>
      <RecordLine type="A" name={domain || "your domain"} value={address.ipv4} />
      {address.ipv6 && <RecordLine type="AAAA" name={domain || "your domain"} value={address.ipv6} />}
    </ul>
  </div>
);
