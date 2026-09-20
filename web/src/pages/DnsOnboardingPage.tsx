import { useNavigate } from "react-router";

import { WizardLayout } from "@/components/auth/WizardLayout";
import { ConnectCloudflareFirst } from "@/components/dns/ConnectCloudflareFirst";
import { DnsSetupStepper } from "@/components/dns/DnsSetupStepper";
import { ErrorDisplay } from "@/components/ErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { Button } from "@/components/ui/button";
import { useFetchConnectorStatus } from "@/hooks/ConnectorsHooks";

// Last owner-wizard step; gates only on the Cloudflare connection, which the Connect step already handled.
export const DnsOnboardingPage = () => {
  const navigate = useNavigate();
  const { data, isPending, error } = useFetchConnectorStatus("cloudflare");
  const connected = data?.status.configured ?? false;

  return (
    <WizardLayout
      step={{ current: 4, total: 4 }}
      title="Set up DNS"
      subtitle="Optional. Give this instance a hostname and choose how traffic reaches it."
    >
      {isPending && <LoadingDisplay />}
      {error && <ErrorDisplay error={error} />}
      {data && !connected && <ConnectCloudflareFirst />}
      {data && connected && <DnsSetupStepper />}
      <div className="mt-10 border-t border-border pt-4">
        <Button variant="link" className="h-auto px-0 text-muted-foreground" onClick={() => navigate("/")}>
          Skip for now
        </Button>
      </div>
    </WizardLayout>
  );
};
