import { useQueryClient } from "@tanstack/react-query";
import { useState } from "react";
import { useNavigate } from "react-router";

import { OwnerWizardRail } from "@/components/auth/OwnerWizardRail";
import { OwnerWizardStepPanel } from "@/components/auth/OwnerWizardStepPanel";
import type { WorkspaceSetupData } from "@/components/auth/SetupWorkspaceStep";
import { WorkspaceReady } from "@/components/auth/WorkspaceReady";
import { ShowcaseSurface } from "@/components/showcase/ShowcaseSurface";
import { WizardShell } from "@/components/wizard/WizardShell";
import { WizardStepHeader } from "@/components/wizard/WizardStepHeader";
import { useCompleteOwnerWizard, useFetchSettings } from "@/hooks/AuthHooks";
import { getWelcomeKey } from "@/hooks/WelcomeHooks";
import { useRenameWorkspace } from "@/hooks/WorkspaceHooks";
import { useOwnerWizardStore } from "@/stores/ownerWizardStore";
import { OWNER_STEPS } from "@/models/OwnerWizard";
import { slugify } from "@/models/Workspace";

const CONFIRM_DELAY_MS = 1600;

// Progress lives in the store, not local state, because step 3's Connect leaves the SPA and remounts this page.
export const OwnerWizardPage = () => {
  const navigate = useNavigate();
  const client = useQueryClient();
  const step = useOwnerWizardStore((s) => s.step);
  const setStep = useOwnerWizardStore((s) => s.setStep);
  const resetProgress = useOwnerWizardStore((s) => s.reset);
  const [confirmed, setConfirmed] = useState(false);
  const { data: settings, isPending: settingsPending, error: settingsError } = useFetchSettings();
  const complete = useCompleteOwnerWizard();
  const renameWorkspace = useRenameWorkspace();

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
    // Progress read before this person became Owner gave no steps; the hand-over must not reuse it.
    await Promise.all([
      new Promise((resolve) => setTimeout(resolve, CONFIRM_DELAY_MS)),
      client.invalidateQueries({ queryKey: [getWelcomeKey], refetchType: "none" }),
    ]);
    navigate("/wizard/welcome", { replace: true });
  };

  const { title, lead } = OWNER_STEPS[step - 1] ?? OWNER_STEPS[0];

  return (
    <>
      {confirmed && <WorkspaceReady />}
      {!confirmed && (
        <ShowcaseSurface quiet>
          <WizardShell rail={<OwnerWizardRail step={step} />}>
            <WizardStepHeader title={title} lead={lead} />
            <div className="mt-8">
              <OwnerWizardStepPanel
                step={step}
                settingsLoading={settingsPending}
                settingsError={settingsError}
                onStep1Continue={() => setStep(2)}
                onStep2Continue={onStep2Continue}
                onFinish={() => void onFinish()}
              />
            </div>
          </WizardShell>
        </ShowcaseSurface>
      )}
    </>
  );
};
