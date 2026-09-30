import { useState } from "react";
import { useNavigate } from "react-router";

import { OwnerWizardStepPanel } from "@/components/auth/OwnerWizardStepPanel";
import { WizardConfirmation } from "@/components/auth/WizardConfirmation";
import { WizardLayout } from "@/components/auth/WizardLayout";
import { useCompleteOwnerWizard, useFetchSettings } from "@/hooks/AuthHooks";
import { useRenameWorkspace, useSelectedWorkspace } from "@/hooks/WorkspaceHooks";
import { useOwnerWizardStore } from "@/stores/ownerWizardStore";
import { slugify, workspacePath } from "@/models/Workspace";

const CONFIRM_DELAY_MS = 900;
const TOTAL_STEPS = 3;
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
  const selected = useSelectedWorkspace();

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
      let slug = selected?.slug ?? "";
      if (workspaceSetup?.workspaceName) {
        // Nothing links to the workspace yet, so naming it here also names its URL.
        const name = workspaceSetup.workspaceName;
        slug = (await renameWorkspace.mutateAsync({ id: workspaceSetup.workspaceId, name, slug: slugify(name) })).slug;
      }
      resetProgress();
      setConfirmed(true);
      // Warm confirmation beat; a workspace starts with no project, so the project wizard is next.
      await new Promise((resolve) => setTimeout(resolve, CONFIRM_DELAY_MS));
      navigate(workspacePath(slug, FIRST_PROJECT_PATH), { replace: true });
    } catch {
      // Error is surfaced by the hook's toast; the step stays open to retry.
      setFinishing(false);
    }
  };

  const { title, subtitle } = STEP_COPY[step - 1] ?? STEP_COPY[0];

  return (
    <WizardLayout step={{ current: step, total: TOTAL_STEPS }} title={title} subtitle={subtitle} onBack={onBack}>
      {confirmed && (
        <WizardConfirmation title="Workspace ready" subtitle="Next, your first project." />
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
