import { useState } from "react";
import { useNavigate } from "react-router";

import { OwnerWizardStepPanel } from "@/components/auth/OwnerWizardStepPanel";
import type { WorkspaceSetupData } from "@/components/auth/SetupWorkspaceStep";
import { WizardLayout } from "@/components/auth/WizardLayout";
import { WorkspaceReady } from "@/components/auth/WorkspaceReady";
import { useCompleteOwnerWizard, useFetchSettings } from "@/hooks/AuthHooks";
import { useRenameWorkspace } from "@/hooks/WorkspaceHooks";
import { useOwnerWizardStore } from "@/stores/ownerWizardStore";
import { slugify } from "@/models/Workspace";

const CONFIRM_DELAY_MS = 1600;
const TOTAL_STEPS = 3;

const STEP_COPY = [
  {
    title: "Introduce yourself",
    subtitle: "You are the first person here, so you become the workspace owner. Tell us who you are.",
  },
  {
    title: "Set up your workspace",
    subtitle: "Name the workspace your team works in. Your first project comes right after.",
  },
  {
    title: "Connect your tools",
    subtitle: "Hook up the services Nexul manages for you. Cloudflare is optional: it is only for your domain, exposures and DNS.",
  },
] as const;

// Progress lives in the store, not local state, because step 3's Connect leaves the SPA and remounts this page.
export const OwnerWizardPage = () => {
  const navigate = useNavigate();
  const step = useOwnerWizardStore((s) => s.step);
  const setStep = useOwnerWizardStore((s) => s.setStep);
  const resetProgress = useOwnerWizardStore((s) => s.reset);
  const [confirmed, setConfirmed] = useState(false);
  const { data: settings, isPending: settingsPending, error: settingsError } = useFetchSettings();
  const complete = useCompleteOwnerWizard();
  const renameWorkspace = useRenameWorkspace();

  const onBack = () => {
    if (step === 1) {
      navigate("/login");
      return;
    }
    setStep(step - 1);
  };

  // Step 3 connects tools, which needs the owner's permissions, so the owner is made here.
  const onStep2Continue = async ({ workspaceId, workspaceName }: WorkspaceSetupData) => {
    if (!settings) return;
    try {
      await complete.mutateAsync(settings.instance_url);
      // Nothing links to the workspace yet, so naming it here also names its URL.
      await renameWorkspace.mutateAsync({ id: workspaceId, name: workspaceName, slug: slugify(workspaceName) });
      setStep(3);
    } catch {
      // Error is surfaced by the hook's toast; the step stays open to retry.
    }
  };

  // The Welcome wizard opens on the owner's first step not done, and ends in the project wizard while there's no project.
  const onFinish = async () => {
    resetProgress();
    setConfirmed(true);
    await new Promise((resolve) => setTimeout(resolve, CONFIRM_DELAY_MS));
    navigate("/wizard/welcome", { replace: true });
  };

  const { title, subtitle } = STEP_COPY[step - 1] ?? STEP_COPY[0];

  return (
    <>
      {confirmed && <WorkspaceReady />}
      {!confirmed && (
        <WizardLayout step={{ current: step, total: TOTAL_STEPS }} title={title} subtitle={subtitle} onBack={onBack}>
          <OwnerWizardStepPanel
            step={step}
            settingsLoading={settingsPending}
            settingsError={settingsError}
            onStep1Continue={() => setStep(2)}
            onStep2Continue={onStep2Continue}
            onFinish={() => void onFinish()}
          />
        </WizardLayout>
      )}
    </>
  );
};
