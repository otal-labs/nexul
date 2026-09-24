import { CommandBlock } from "@/components/pairing/CommandBlock";
import { ErrorDisplay } from "@/components/ErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { useFetchTunnelToken } from "@/hooks/PairingHooks";
import {
  detectTunnelOs,
  TUNNEL_OS_LABELS,
  TunnelOs,
  tunnelInstallSteps,
  type TunnelInstallStep,
} from "@/utils/TunnelInstallCommands";

interface InstallStepItemProps {
  step: TunnelInstallStep;
  number: number;
}

const InstallStepItem = ({ step, number }: InstallStepItemProps) => (
  <li className="space-y-2">
    <p className="text-sm">
      <span className="mr-1.5 font-mono text-muted-foreground tabular-nums">{number}.</span>
      {step.title}
    </p>
    <CommandBlock shell={step.shell} lines={step.lines} />
  </li>
);

interface InstallStepListProps {
  os: TunnelOs;
  token: string;
}

const InstallStepList = ({ os, token }: InstallStepListProps) => (
  <ol className="space-y-4">
    {tunnelInstallSteps(os, token).map((step, i) => (
      <InstallStepItem key={step.title} step={step} number={i + 1} />
    ))}
  </ol>
);

interface TunnelInstallStepsProps {
  computerId: string;
}

// The per-OS commands that install cloudflared as a service with this computer's connector token.
export const TunnelInstallSteps = ({ computerId }: TunnelInstallStepsProps) => {
  const { data: token, error, isPending } = useFetchTunnelToken(computerId);
  return (
    <div className="min-w-0 space-y-3">
      <p className="text-sm text-muted-foreground">Run these on the computer you're pairing. The token is secret, so keep it to that computer.</p>
      {isPending && <LoadingDisplay />}
      {error && <ErrorDisplay error={error} />}
      {token && (
        <Tabs defaultValue={detectTunnelOs(navigator.userAgent)}>
          <TabsList className="w-full">
            {Object.values(TunnelOs).map((os) => (
              <TabsTrigger key={os} value={os}>
                {TUNNEL_OS_LABELS[os]}
              </TabsTrigger>
            ))}
          </TabsList>
          {Object.values(TunnelOs).map((os) => (
            <TabsContent key={os} value={os} className="pt-2">
              <InstallStepList os={os} token={token} />
            </TabsContent>
          ))}
        </Tabs>
      )}
    </div>
  );
};
