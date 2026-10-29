import { SignedOutFrame } from "@/components/auth/SignedOutFrame";
import { SignedOutHeader } from "@/components/auth/SignedOutHeader";
import { StepSegments } from "@/components/auth/StepSegments";
import { GitHubAppForm } from "@/components/setup/GitHubAppForm";
import { SetupCodeForm } from "@/components/setup/SetupCodeForm";
import { SetupDomainStep } from "@/components/setup/SetupDomainStep";
import { SetupDone } from "@/components/setup/SetupDone";
import { SetupHandoff } from "@/components/setup/SetupHandoff";
import { SetupLocalFinish } from "@/components/setup/SetupLocalFinish";
import { useSetupPassStore } from "@/stores/setupPassStore";
import { SetupStages, setupStage, setupStageCopy } from "@/models/Setup";
import type { BootstrapStatus } from "@/models/User";

interface SetupFlowProps {
  status: BootstrapStatus;
}

const SETUP_STEPS = 3;

// Each stage is derived from bootstrap-status, the pass and this origin, so a reload lands on the same screen.
export const SetupFlow = ({ status }: SetupFlowProps) => {
  const hasPass = useSetupPassStore((s) => s.token !== null);
  const stage = setupStage(status, hasPass, window.location.origin);
  const copy = setupStageCopy[stage];
  const instanceUrl = status.instance_url ?? "";

  return (
    <SignedOutFrame quiet>
      <SignedOutHeader
        title={copy.title}
        subtitle={copy.subtitle}
        progress={copy.step > 0 && <StepSegments current={copy.step} total={SETUP_STEPS} />}
      />
      <div className="mt-6">
        {stage === SetupStages.Done && <SetupDone />}
        {stage === SetupStages.Code && <SetupCodeForm />}
        {stage === SetupStages.Local && <SetupLocalFinish />}
        {stage === SetupStages.Domain && <SetupDomainStep />}
        {stage === SetupStages.Handoff && <SetupHandoff instanceUrl={instanceUrl} />}
        {stage === SetupStages.GitHub && <GitHubAppForm instanceUrl={instanceUrl} />}
      </div>
    </SignedOutFrame>
  );
};
