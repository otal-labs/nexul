import type { ReactNode } from "react";
import { useNavigate } from "react-router";

import { Button } from "@/components/ui/button";
import { DnsRungBody } from "@/components/dns/DnsRungBody";
import { DnsSetupRail } from "@/components/dns/DnsSetupRail";
import { WizardShell } from "@/components/wizard/WizardShell";
import { WizardStepHeader } from "@/components/wizard/WizardStepHeader";
import { useDnsSetup } from "@/hooks/useDnsSetup";

interface DnsSetupWizardProps {
  // What stands in for the rungs while setup can't start (Cloudflare not connected, a role that can't write DNS).
  gate?: ReactNode;
}

// Optional DNS setup after first run, one rung on screen at a time beside the rail of all of them.
export const DnsSetupWizard = ({ gate }: DnsSetupWizardProps) => {
  const navigate = useNavigate();
  const setup = useDnsSetup();
  const rung = setup.rungs.find((r) => r.key === setup.view);
  const title = gate ? "Set up DNS" : (rung?.title ?? "Set up DNS");
  const lead = gate ? "Optional. Give this instance a hostname and choose how traffic reaches it." : rung?.lead;

  return (
    <WizardShell
      rail={<DnsSetupRail setup={setup} />}
      exit={
        <Button variant="ghost" size="sm" className="-mr-2 h-7 px-2 text-muted-foreground" onClick={() => navigate("/")}>
          Skip for now
        </Button>
      }
    >
      <WizardStepHeader title={title} lead={lead} />
      <div className="mt-8">
        {gate}
        {!gate && <DnsRungBody setup={setup} />}
      </div>
    </WizardShell>
  );
};
