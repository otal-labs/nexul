import { ConnectToolsStep } from "@/components/auth/ConnectToolsStep";
import { IntroduceYourselfStep } from "@/components/auth/IntroduceYourselfStep";
import { SetupWorkspaceStep } from "@/components/auth/SetupWorkspaceStep";
import type { WorkspaceSetupData } from "@/components/auth/SetupWorkspaceStep";
import { ErrorDisplay } from "@/components/ErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";

interface OwnerWizardStepPanelProps {
  step: number;
  settingsLoading: boolean;
  settingsError: unknown;
  onStep1Continue: () => void;
  onStep2Continue: (data: WorkspaceSetupData) => Promise<void>;
  onFinish: () => void;
}

export const OwnerWizardStepPanel = ({
  step,
  settingsLoading,
  settingsError,
  onStep1Continue,
  onStep2Continue,
  onFinish,
}: OwnerWizardStepPanelProps) => (
  <>
    {step === 1 && <IntroduceYourselfStep onContinue={onStep1Continue} />}
    {step === 2 && settingsLoading && <LoadingDisplay />}
    {step === 2 && settingsError && <ErrorDisplay error={settingsError} />}
    {step === 2 && !settingsLoading && !settingsError && <SetupWorkspaceStep onContinue={onStep2Continue} />}
    {step === 3 && <ConnectToolsStep onContinue={onFinish} />}
  </>
);
