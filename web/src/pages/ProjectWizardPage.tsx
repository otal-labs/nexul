import { useEffect } from "react";
import { Navigate, useParams, useSearchParams } from "react-router";

import { WizardLayout } from "@/components/auth/WizardLayout";
import { ProjectWizardStepper, WizardSteps, type WizardStepId } from "@/components/wizard/ProjectWizardStepper";
import { useFetchProject } from "@/hooks/ProjectHooks";
import { useFetchStack } from "@/hooks/StackHooks";
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

// The store is not persisted, so a reload on a later rung lands with nothing to show; this names the furthest
// rung the store can still render, and the page falls back to it. Door 2's ?project= and door 3's ?stack= both
// count for the repository rung because their seed arrives asynchronously.
const furthestStep = (hasProjectParam: boolean, hasStackParam: boolean): WizardStepId => {
  const { projectId, candidate, stackId } = useProjectWizardStore.getState();
  if (stackId) return "done";
  if (candidate) return "service";
  if (projectId || hasProjectParam || hasStackParam) return "repository";
  return "project";
};

const wizardTitle = (isAttach: boolean, projectPreselected: boolean): string => {
  if (isAttach) return "Attach a repository";
  if (projectPreselected) return "Add a service";
  return "New project";
};

// An unknown step goes to the first one; a step past the furthest reachable one goes back to that one.
const wizardRedirect = (step: string | undefined, searchParams: URLSearchParams, isAttach: boolean): string | null => {
  if (!isWizardStep(step)) return "/wizard/project/project";
  const allowed = furthestStep(searchParams.has("project"), isAttach);
  if (WizardSteps.indexOf(step) <= WizardSteps.indexOf(allowed)) return null;
  const query = searchParams.toString();
  return `/wizard/project/${allowed}${query ? `?${query}` : ""}`;
};

export const ProjectWizardPage = () => {
  const { step } = useParams<{ step: string }>();
  const [searchParams] = useSearchParams();
  useSeedPreselectedProject();
  useSeedAttachStack();
  const projectPreselected = useProjectWizardStore((s) => s.projectPreselected);
  const isAttach = searchParams.has("stack");

  const redirectTo = wizardRedirect(step, searchParams, isAttach);
  const title = wizardTitle(isAttach, projectPreselected);
  const subtitle = isAttach
    ? "Point this stack at a repository so Nexul can build and deploy it."
    : "Point Nexul at a repository and it takes care of the rest.";

  return (
    <>
      {redirectTo && <Navigate to={redirectTo} replace />}
      {!redirectTo && isWizardStep(step) && (
        <WizardLayout
          step={{ current: WizardSteps.indexOf(step) + 1, total: WizardSteps.length }}
          title={title}
          subtitle={subtitle}
        >
          <ProjectWizardStepper step={step} />
        </WizardLayout>
      )}
    </>
  );
};
