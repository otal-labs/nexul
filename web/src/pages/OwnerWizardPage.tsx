import { useState } from "react";
import { useNavigate } from "react-router";

import { OwnerWizardStepPanel } from "@/components/auth/OwnerWizardStepPanel";
import type { WorkspaceSetupData } from "@/components/auth/SetupWorkspaceStep";
import { WizardConfirmation } from "@/components/auth/WizardConfirmation";
import { WizardLayout } from "@/components/auth/WizardLayout";
import { useCompleteOwnerWizard, useFetchSettings } from "@/hooks/AuthHooks";
import { useRenameWorkspace, useSelectedWorkspace } from "@/hooks/WorkspaceHooks";
import { useOwnerWizardStore } from "@/stores/ownerWizardStore";
import { slugify, workspacePath } from "@/models/Workspace";

const CONFIRM_DELAY_MS = 900;
const TOTAL_STEPS = 4;
const FIRST_PROJECT_PATH = "/wizard/project/project";

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
    subtitle: "Hook up the services Nexul manages for you.",
  },
  {
    title: "Set up T3 Code",
    subtitle: "Agents run on your own computer through T3 Code. Add it with one command, and Nexul pairs it for you.",
  },
] as const;

// Progress lives in the store, not local state, because step 3's Connect leaves the SPA and remounts this page.
export const OwnerWizardPage = () => {
  const navigate = useNavigate();
  const step = useOwnerWizardStore((s) => s.step);
  const setStep = useOwnerWizardStore((s) => s.setStep);
  const workspaceSlug = useOwnerWizardStore((s) => s.workspaceSlug);
  const setWorkspaceSlug = useOwnerWizardStore((s) => s.setWorkspaceSlug);
  const resetProgress = useOwnerWizardStore((s) => s.reset);
  const [confirmed, setConfirmed] = useState(false);
  const { data: settings, isPending: settingsPending, error: settingsError } = useFetchSettings();
  const complete = useCompleteOwnerWizard();
  const renameWorkspace = useRenameWorkspace();
  const selected = useSelectedWorkspace();

  const onBack = () => {
    if (step === 1) {
      navigate("/login");
      return;
    }
    setStep(step - 1);
  };

  // Steps 3 and 4 connect tools and add a computer, which need the owner's permissions, so the owner is made here.
  const onStep2Continue = async ({ workspaceId, workspaceName }: WorkspaceSetupData) => {
    if (!settings) return;
    try {
      await complete.mutateAsync(settings.instance_url);
      // Nothing links to the workspace yet, so naming it here also names its URL.
      const renamed = await renameWorkspace.mutateAsync({ id: workspaceId, name: workspaceName, slug: slugify(workspaceName) });
      setWorkspaceSlug(renamed.slug);
      setStep(3);
    } catch {
      // Error is surfaced by the hook's toast; the step stays open to retry.
    }
  };

  const onFinish = async () => {
    const slug = workspaceSlug ?? selected?.slug ?? "";
    resetProgress();
    setConfirmed(true);
    // Warm confirmation beat; a workspace starts with no project, so the project wizard is next.
    await new Promise((resolve) => setTimeout(resolve, CONFIRM_DELAY_MS));
    navigate(workspacePath(slug, FIRST_PROJECT_PATH), { replace: true });
  };

  const { title, subtitle } = STEP_COPY[step - 1] ?? STEP_COPY[0];

  return (
    <WizardLayout step={{ current: step, total: TOTAL_STEPS }} title={title} subtitle={subtitle} onBack={onBack}>
      {confirmed && <WizardConfirmation title="Workspace ready" subtitle="Next, your first project." />}
      {!confirmed && (
        <OwnerWizardStepPanel
          step={step}
          settingsLoading={settingsPending}
          settingsError={settingsError}
          onStep1Continue={() => setStep(2)}
          onStep2Continue={onStep2Continue}
          onStep3Continue={() => setStep(4)}
          onFinish={() => void onFinish()}
        />
      )}
    </WizardLayout>
  );
};
