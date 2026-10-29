import { Check } from "lucide-react";

import { Button } from "@/components/ui/button";
import { DnsSetupDone } from "@/components/dns/DnsSetupDone";
import { EntryPathChoice } from "@/components/dns/EntryPathChoice";
import { InstanceRecordForm } from "@/components/dns/InstanceRecordForm";
import { ProxyEntryPathForm } from "@/components/dns/ProxyEntryPathForm";
import { TunnelDeployStep } from "@/components/dns/TunnelDeployStep";
import { TunnelHostnameStep } from "@/components/dns/TunnelHostnameStep";
import { InWizardBar } from "@/components/wizard/InWizardBar";
import type { DnsSetup } from "@/hooks/useDnsSetup";
import { EntryPaths } from "@/models/DNS";

interface DnsRungBodyProps {
  setup: DnsSetup;
}

// The rung on screen: its form while it is the next to do, or what it set up, with Change on the path that everything hangs off.
export const DnsRungBody = ({ setup }: DnsRungBodyProps) => {
  const { rungs, view, path, tunnel, result, open, choose, connect, finish, unfinish, restart } = setup;
  const rung = rungs.find((r) => r.key === view);
  const done = rung?.state === "done";

  return (
    <>
      {done && (
        <div className="flex items-start gap-3 rounded-lg border border-border bg-card/60 px-4 py-3">
          <Check className="mt-0.5 size-4 shrink-0 text-success" aria-hidden />
          <p className="min-w-0 flex-1 text-sm">{rung.summary}</p>
          {view === "path" && (
            <Button variant="ghost" size="sm" className="-my-1 -mr-2 text-muted-foreground" onClick={restart}>
              Change
            </Button>
          )}
        </div>
      )}
      {done && (
        <InWizardBar>
          <Button onClick={() => open(null)}>Continue</Button>
        </InWizardBar>
      )}
      {!done && view === "path" && <EntryPathChoice onContinue={choose} />}
      {!done && view === "tunnel" && <TunnelDeployStep onConnected={connect} />}
      {!done && view === "hostname" && path === EntryPaths.Bare && <InstanceRecordForm onDone={finish} />}
      {!done && view === "hostname" && tunnel && <TunnelHostnameStep deployment={tunnel} onDone={finish} />}
      {!done && view === "hostname" && path === EntryPaths.Proxy && <ProxyEntryPathForm onDone={finish} />}
      {view === "live" && result && <DnsSetupDone result={result} onRecordDeleted={unfinish} />}
    </>
  );
};
