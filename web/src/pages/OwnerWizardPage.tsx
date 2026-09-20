import { useState } from "react";
import { useNavigate } from "react-router";

import { OwnerWizardStepPanel } from "@/components/auth/OwnerWizardStepPanel";
import { WizardConfirmation } from "@/components/auth/WizardConfirmation";
import { WizardLayout } from "@/components/auth/WizardLayout";
import { useCompleteOwnerWizard, useFetchSettings } from "@/hooks/AuthHooks";
import { useRenameProject, useSetProjectPrefix } from "@/hooks/ProjectHooks";
import { useRenameWorkspace } from "@/hooks/WorkspaceHooks";
import { useOwnerWizardStore } from "@/stores/ownerWizardStore";

const CONFIRM_DELAY_MS = 900;
// DNS onboarding is the fourth rung, rendered by DnsOnboardingPage after the confirmation beat.
const TOTAL_STEPS = 4;

const STEP_COPY = [
  {
    title: "Introduce yourself",
    subtitle: "You are the first person here, so you become the workspace owner. Tell us who you are.",
  },
  {
    title: "Set up your workspace",
    subtitle: "Name your workspace and its default project.",
  },
  {
    title: "Connect your tools",
    subtitle: "Hook up the services Nexul manages for you.",
  },
] as const;

// Progress lives in the store, not local state, because step 3's Connect leaves the SPA and remounts this page.
export const OwnerWizardPage = () => {
  const navigate = useNavigate();
  const step = useOwnerWizardStore((s) => s.step);
  const setStep = useOwnerWizardStore((s) => s.setStep);
  const workspaceSetup = useOwnerWizardStore((s) => s.workspaceSetup);
  const setWorkspaceSetup = useOwnerWizardStore((s) => s.setWorkspaceSetup);
  const resetProgress = useOwnerWizardStore((s) => s.reset);
  const [confirmed, setConfirmed] = useState(false);
  // Covers the whole multi-mutation finish sequence, so a second click mid-sequence can't race itself.
  const [finishing, setFinishing] = useState(false);
  const { data: settings, isPending: settingsPending, error: settingsError } = useFetchSettings();
  const complete = useCompleteOwnerWizard();
  const renameWorkspace = useRenameWorkspace();
  const renameProject = useRenameProject();
  const setPrefix = useSetProjectPrefix();

  const onBack = () => {
    if (step === 1) {
      navigate("/login");
      return;
    }
    setStep(step - 1);
  };

  const onFinish = async () => {
    if (!settings || finishing) return;
    setFinishing(true);
    try {
      await complete.mutateAsync(settings.instance_url);
      if (workspaceSetup) {
        if (workspaceSetup.workspaceName) {
          await renameWorkspace.mutateAsync({ id: workspaceSetup.workspaceId, name: workspaceSetup.workspaceName });
        }
        await renameProject.mutateAsync({ id: workspaceSetup.projectId, name: workspaceSetup.projectName });
        await setPrefix.mutateAsync({ id: workspaceSetup.projectId, prefix: workspaceSetup.projectPrefix });
      }
      resetProgress();
      setConfirmed(true);
      // Warm confirmation beat before the optional DNS step.
      await new Promise((resolve) => setTimeout(resolve, CONFIRM_DELAY_MS));
      navigate("/wizard/onboarding/dns", { replace: true });
    } catch {
      // Error is surfaced by the hook's toast; the step stays open to retry.
      setFinishing(false);
    }
  };

  const { title, subtitle } = STEP_COPY[step - 1] ?? STEP_COPY[0];

  return (
    <WizardLayout step={{ current: step, total: TOTAL_STEPS }} title={title} subtitle={subtitle} onBack={onBack}>
      {confirmed && (
        <WizardConfirmation
          title="Workspace ready"
          subtitle="One more step — we will point your hostname at the internet."
        />
      )}
      {!confirmed && (
        <OwnerWizardStepPanel
          step={step}
          settingsLoading={settingsPending}
          settingsError={settingsError}
          finishing={finishing}
          onStep1Continue={() => setStep(2)}
          onStep2Continue={(data) => {
            setWorkspaceSetup(data);
            setStep(3);
          }}
          onFinish={() => void onFinish()}
        />
      )}
    </WizardLayout>
  );
};
