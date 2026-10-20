import { useEffect } from "react";
import { Navigate, useParams, useSearchParams } from "react-router";

import { ProjectWizardFrame } from "@/components/wizard/ProjectWizardFrame";
import { ProjectWizardStepContent } from "@/components/wizard/ProjectWizardStepContent";
import { useFetchProject } from "@/hooks/ProjectHooks";
import { useFetchStack } from "@/hooks/StackHooks";
import { useSeedScan } from "@/hooks/useWaitingRepos";
import { useWorkspacePath } from "@/hooks/useWorkspacePath";
import { WizardSteps, type WizardStepId } from "@/models/ProjectWizard";
import { useProjectWizardStore } from "@/stores/projectWizardStore";

const isWizardStep = (value: string | undefined): value is WizardStepId =>
  !!value && (WizardSteps as readonly string[]).includes(value);

// Door 2 ("Add service") arrives with ?project=<id> and starts at the repository step; this seeds the store
// from it once so the project rung can render pre-completed without the owner re-entering anything.
const useSeedPreselectedProject = () => {
  const [searchParams] = useSearchParams();
  const paramProjectId = searchParams.get("project") ?? undefined;
  const projectId = useProjectWizardStore((s) => s.projectId);
  const setProjectId = useProjectWizardStore((s) => s.setProjectId);
  const { data: project } = useFetchProject(paramProjectId && paramProjectId !== projectId ? paramProjectId : undefined);

  useEffect(() => {
    if (project && project.id !== projectId) setProjectId(project.id, project.name, true);
  }, [project, projectId, setProjectId]);
};

// Door 3 ("Attach repository") arrives with ?stack=<id> from an unmanaged stack's header button; this fetches
// the stack for its project_id, seeds the project the same way door 2 does, and records which stack the wizard
// is attaching so the service rung can skip name/machine and PATCH instead of creating.
const useSeedAttachStack = () => {
  const [searchParams] = useSearchParams();
  const paramStackId = searchParams.get("stack") ?? undefined;
  const projectId = useProjectWizardStore((s) => s.projectId);
  const attachStackId = useProjectWizardStore((s) => s.attachStackId);
  const setProjectId = useProjectWizardStore((s) => s.setProjectId);
  const setAttachStackId = useProjectWizardStore((s) => s.setAttachStackId);
  // Stays enabled on the stable param: the project fetch is chained off this data, and disabling the query
  // once attachStackId is set would drop the data to undefined before the chained fetch resolves.
  const { data: stack } = useFetchStack(paramStackId);
  const { data: project } = useFetchProject(stack && stack.project_id !== projectId ? stack.project_id : undefined);

  // Reset only runs from the done rung, so an abandoned attach must not leak its stack into the next door.
  useEffect(() => {
    if (!paramStackId && attachStackId) setAttachStackId(null);
    if (stack && stack.id !== attachStackId) setAttachStackId(stack.id);
  }, [paramStackId, stack, attachStackId, setAttachStackId]);

  useEffect(() => {
    if (project && project.id !== projectId) setProjectId(project.id, project.name, true);
  }, [project, projectId, setProjectId]);
};

export const ProjectWizardPage = () => {
  const { step } = useParams<{ step: string }>();
  useSeedPreselectedProject();
  const wsPath = useWorkspacePath();
  useSeedAttachStack();
  useSeedScan();
  const reset = useProjectWizardStore((s) => s.reset);
  // Leaving ends the run, so the next visit starts clean; a project it already made resumes through Continue setup.
  useEffect(() => reset, [reset]);

  return (
    <>
      {!isWizardStep(step) && <Navigate to={wsPath("/wizard/project/project")} replace />}
      {isWizardStep(step) && (
        <ProjectWizardFrame step={step}>
          <ProjectWizardStepContent step={step} />
        </ProjectWizardFrame>
      )}
    </>
  );
};
