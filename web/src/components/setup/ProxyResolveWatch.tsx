import { CheckCircle2, Loader2 } from "lucide-react";
import { useEffect, useRef } from "react";

import { ProxyRecordLines } from "@/components/setup/ProxyRecordLines";
import { useResolveHost } from "@/hooks/DnsHooks";
import { resolvesHere, type PublicAddress } from "@/models/Setup";

interface ProxyResolveWatchProps {
  domain: string;
  address: PublicAddress;
  onResolved: () => void;
}

// Let's Encrypt only issues once the name points here, so nothing deploys before every answer is this server.
export const ProxyResolveWatch = ({ domain, address, onResolved }: ProxyResolveWatchProps) => {
  const { data } = useResolveHost(domain, address);
  const resolved = resolvesHere(data?.addresses, address);
  const answers = data?.addresses ?? [];
  const fired = useRef(false);

  // Side effect on an external event (DNS starting to answer with this server), fired once.
  useEffect(() => {
    if (!resolved || fired.current) return;
    fired.current = true;
    onResolved();
  }, [resolved, onResolved]);

  return (
    <div className="space-y-4">
      <ProxyRecordLines domain={domain} address={address} />
      <p role="status" className="flex items-start gap-2 text-sm">
        {!resolved && <Loader2 className="mt-0.5 size-4 shrink-0 animate-spin motion-reduce:animate-none" aria-hidden />}
        {resolved && <CheckCircle2 className="mt-0.5 size-4 shrink-0 text-success" aria-hidden />}
        <span className="min-w-0">
          {resolved && `${domain} points at this server.`}
          {!resolved && `Waiting for ${domain} to point at this server…`}
          {!resolved && (
            <span className="block break-all font-mono text-xs text-muted-foreground">
              {answers.length > 0 ? `resolves to ${answers.join(", ")} now` : "does not resolve yet"}
            </span>
          )}
        </span>
      </p>
    </div>
  );
};
