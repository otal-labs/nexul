import { Button } from "@/components/ui/button";
import { TickerRow, type CheckOutcome } from "@/components/TickerRow";
import { TUNNEL_ROUTE_CHECKS } from "@/models/DNS";

interface TunnelRouteChecksProps {
  hostname: string;
  outcomeFor: (key: string) => CheckOutcome;
  verified: boolean;
  verifying: boolean;
  onCheckAgain: () => void;
  onContinue: () => void;
}

// The ticker after routing: each row lights as its check lands, and Continue waits until the hostname answers.
export const TunnelRouteChecks = ({
  hostname,
  outcomeFor,
  verified,
  verifying,
  onCheckAgain,
  onContinue,
}: TunnelRouteChecksProps) => (
  <div className="space-y-4">
    <p className="font-mono text-xs text-muted-foreground">{hostname}</p>
    <ul className="space-y-3">
      {TUNNEL_ROUTE_CHECKS.map((c) => (
        <TickerRow key={c.key} label={c.label} why={c.why} outcome={outcomeFor(c.key)} />
      ))}
    </ul>
    {verified && (
      <Button className="w-full sm:w-auto" onClick={onContinue}>
        Continue
      </Button>
    )}
    {!verified && !verifying && (
      <Button variant="outline" className="w-full sm:w-auto" onClick={onCheckAgain}>
        Check again
      </Button>
    )}
  </div>
);
