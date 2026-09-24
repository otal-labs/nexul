import { ExternalLink, Info } from "lucide-react";
import { Link } from "react-router";

import { Button } from "@/components/ui/button";
import type { TunnelPrerequisite } from "@/models/Pairing";

const COPY: Record<TunnelPrerequisite, { title: string; body: string }> = {
  cloudflare_not_connected: {
    title: "Cloudflare isn't connected",
    body: "Each computer gets its own tunnel in this instance's Cloudflare account. Connect Cloudflare in Settings, then try again.",
  },
  zero_trust_disabled: {
    title: "Zero Trust isn't enabled",
    body: "Nexul closes every computer's hostname with Cloudflare Access, which needs Zero Trust. Enable it once in the Cloudflare dashboard: pick a team name and the Free plan. Then try again.",
  },
};

interface TunnelPrerequisiteAlertProps {
  reason: TunnelPrerequisite;
  onRetry: () => void;
  retrying: boolean;
}

// A missing instance prerequisite explained in the step, with its fix, instead of a dead end.
export const TunnelPrerequisiteAlert = ({ reason, onRetry, retrying }: TunnelPrerequisiteAlertProps) => (
  <div role="alert" className="flex gap-3 rounded-lg border border-border bg-card p-4">
    <Info className="mt-0.5 size-4 shrink-0 text-warning" aria-hidden />
    <div className="min-w-0 space-y-3">
      <div className="space-y-1">
        <p className="text-sm font-medium">{COPY[reason].title}</p>
        <p className="text-sm text-muted-foreground">{COPY[reason].body}</p>
      </div>
      <div className="flex flex-wrap gap-2">
        {reason === "cloudflare_not_connected" && (
          <Button asChild size="sm">
            <Link to="/settings?section=connectors">Connect Cloudflare</Link>
          </Button>
        )}
        {reason === "zero_trust_disabled" && (
          <Button asChild size="sm">
            <a href="https://one.dash.cloudflare.com/" target="_blank" rel="noreferrer">
              Open Zero Trust
              <ExternalLink className="size-3.5" aria-hidden />
            </a>
          </Button>
        )}
        <Button type="button" size="sm" variant="outline" onClick={onRetry} disabled={retrying}>
          Try again
        </Button>
      </div>
    </div>
  </div>
);
