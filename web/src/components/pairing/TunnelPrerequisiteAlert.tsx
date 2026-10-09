import { ExternalLink, Info } from "lucide-react";
import { Link } from "react-router";

import { Button } from "@/components/ui/button";
import { DialogClose } from "@/components/ui/dialog";
import { useCanOpenSection } from "@/hooks/AccessHooks";
import { OWNER_WIZARD_COMPLETED_STEP, OWNER_WIZARD_TOOLS_STEP, useOwnerWizardStore } from "@/stores/ownerWizardStore";
import type { TunnelPrerequisite } from "@/models/Pairing";

const COPY: Record<TunnelPrerequisite, { title: string; body: string }> = {
  cloudflare_not_connected: {
    title: "Cloudflare isn't connected",
    body: "Each computer gets its own tunnel in this instance's Cloudflare account. Connect Cloudflare in Settings, then try again.",
  },
  zero_trust_disabled: {
    title: "Zero Trust isn't enabled",
    body: "Nexul puts every computer's hostname behind Cloudflare Access, which needs Zero Trust. Enable it once in the Cloudflare dashboard with a team name and the Free plan, then try again.",
  },
};

interface TunnelPrerequisiteAlertProps {
  reason: TunnelPrerequisite;
  onRetry: () => void;
  retrying: boolean;
}

// A missing instance prerequisite explained in the step, with its fix, instead of a dead end.
export const TunnelPrerequisiteAlert = ({ reason, onRetry, retrying }: TunnelPrerequisiteAlertProps) => {
  const canConnect = useCanOpenSection("connectors");
  // In the owner wizard Settings is out of reach until it finishes, so Cloudflare is connected on its previous step.
  const inOwnerWizard = useOwnerWizardStore((s) => s.step > OWNER_WIZARD_COMPLETED_STEP);
  const setWizardStep = useOwnerWizardStore((s) => s.setStep);
  const connectCloudflare = reason === "cloudflare_not_connected" && canConnect;
  return (
    <div role="alert" className="flex gap-3 rounded-lg border border-border bg-card p-4">
      <Info className="mt-0.5 size-4 shrink-0 text-warning" aria-hidden />
      <div className="min-w-0 space-y-3">
        <div className="space-y-1">
          <p className="text-sm font-medium">{COPY[reason].title}</p>
          <p className="text-sm text-muted-foreground">{COPY[reason].body}</p>
        </div>
        <div className="flex flex-wrap gap-2">
          {connectCloudflare && inOwnerWizard && (
            <DialogClose asChild>
              <Button type="button" size="sm" onClick={() => setWizardStep(OWNER_WIZARD_TOOLS_STEP)}>
                Connect Cloudflare
              </Button>
            </DialogClose>
          )}
          {connectCloudflare && !inOwnerWizard && (
            <Button asChild size="sm">
              <Link to="/settings/connectors">Connect Cloudflare</Link>
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
          <Button type="button" size="sm" variant="outline" onClick={onRetry} loading={retrying}>
            Try again
          </Button>
        </div>
      </div>
    </div>
  );
};
