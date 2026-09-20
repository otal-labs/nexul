import { useState } from "react";

import { DnsSetupDone } from "@/components/dns/DnsSetupDone";
import { DnsStep, type DnsStepState } from "@/components/dns/DnsStep";
import { EntryPathChoice } from "@/components/dns/EntryPathChoice";
import { InstanceRecordForm } from "@/components/dns/InstanceRecordForm";
import { ProxyEntryPathForm } from "@/components/dns/ProxyEntryPathForm";
import { TunnelDeployStep } from "@/components/dns/TunnelDeployStep";
import { TunnelHostnameStep } from "@/components/dns/TunnelHostnameStep";
import { EntryPaths, entryPathOptions, type DnsSetupResult, type EntryPath, type TunnelDeployment } from "@/models/DNS";

const stepState = (unlocked: boolean, done: boolean): DnsStepState => {
  if (!unlocked) return "upcoming";
  if (done) return "done";
  return "active";
};

// Three rungs, one unlocked at a time (Resend "Add domain" lock): choose the path, point the hostname, go live.
export const DnsSetupStepper = () => {
  const [path, setPath] = useState<EntryPath | null>(null);
  const [tunnel, setTunnel] = useState<TunnelDeployment | null>(null);
  const [result, setResult] = useState<DnsSetupResult | null>(null);
  const chosen = entryPathOptions.find((option) => option.value === path);
  // The tunnel path deploys cloudflared before a hostname can be routed into it; the other paths go straight to the record.
  const isTunnel = path === EntryPaths.Tunnel;
  const hostnameUnlocked = path !== null && (!isTunnel || tunnel !== null);

  const restart = () => {
    setPath(null);
    setTunnel(null);
    setResult(null);
  };

  return (
    <ol className="list-none">
      <DnsStep
        title="How should traffic reach this instance?"
        description="Nexul deploys your chosen entry path itself, like any other service."
        state={stepState(true, path !== null)}
        summary={chosen?.label}
        onChange={restart}
      >
        <EntryPathChoice onContinue={setPath} />
      </DnsStep>
      {isTunnel && (
        <DnsStep
          title="Deploy the tunnel"
          description="Nexul creates the tunnel at Cloudflare and runs cloudflared on your runner until it connects."
          state={stepState(true, tunnel !== null)}
          summary={tunnel && `Tunnel ${tunnel.tunnelName} connected · cloudflared on ${tunnel.target}`}
        >
          <TunnelDeployStep onConnected={setTunnel} />
        </DnsStep>
      )}
      <DnsStep
        title="Point your hostname"
        description={chosen?.stepDescription}
        state={stepState(hostnameUnlocked, result !== null)}
        summary={result?.headline}
      >
        {path === EntryPaths.Bare && <InstanceRecordForm onDone={setResult} />}
        {isTunnel && tunnel && <TunnelHostnameStep deployment={tunnel} onDone={setResult} />}
        {path === EntryPaths.Proxy && <ProxyEntryPathForm onDone={setResult} />}
      </DnsStep>
      <DnsStep title="Go live" state={stepState(result !== null, false)} last>
        {result && <DnsSetupDone result={result} />}
      </DnsStep>
    </ol>
  );
};
