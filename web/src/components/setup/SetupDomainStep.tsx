import { useState } from "react";

import { DnsStep } from "@/components/dns/DnsStep";
import { EntryPathChoice } from "@/components/dns/EntryPathChoice";
import { SetupFinishStatus } from "@/components/setup/SetupFinishStatus";
import { SetupOwnHttpsForm } from "@/components/setup/SetupOwnHttpsForm";
import { SetupProxyPath } from "@/components/setup/SetupProxyPath";
import { SetupTunnelRungs } from "@/components/setup/SetupTunnelRungs";
import { useSetInstanceUrl } from "@/hooks/SetupHooks";
import { EntryPaths, setupEntryPathOptions, stepState, type EntryPath } from "@/models/DNS";

// No skip: GitHub sign-in needs the final https address, so every path ends in the same stored-URL check.
export const SetupDomainStep = () => {
  const [path, setPath] = useState<EntryPath | null>(null);
  const finish = useSetInstanceUrl();
  const chosen = setupEntryPathOptions.find((option) => option.value === path);
  const finishing = !finish.isIdle;

  const onFinish = (url: string, attempts = 1) => finish.mutate({ url, attempts });

  const restart = () => {
    finish.reset();
    setPath(null);
  };

  return (
    <ol className="list-none">
      <DnsStep
        title="How should traffic reach Nexul?"
        description="Pick the one that matches this server. You can change it later from the DNS page."
        state={stepState(true, path !== null)}
        summary={chosen?.label}
        onChange={restart}
      >
        <EntryPathChoice options={setupEntryPathOptions} onContinue={setPath} />
      </DnsStep>
      {path === EntryPaths.Tunnel && <SetupTunnelRungs onFinish={onFinish} />}
      {path === EntryPaths.Proxy && (
        <DnsStep
          title="Point your domain here"
          description={chosen?.stepDescription}
          state={stepState(true, finishing)}
          summary={finish.variables?.url}
        >
          <SetupProxyPath onFinish={onFinish} />
        </DnsStep>
      )}
      {path === EntryPaths.OwnHttps && (
        <DnsStep
          title="Your HTTPS address"
          description={chosen?.stepDescription}
          state={stepState(true, finishing)}
          summary={finish.variables?.url}
          onChange={() => finish.reset()}
        >
          <SetupOwnHttpsForm onFinish={onFinish} />
        </DnsStep>
      )}
      <DnsStep title="Go live" state={stepState(finishing, false)} last>
        <SetupFinishStatus finish={finish} />
      </DnsStep>
    </ol>
  );
};
