import { displayTitleClass } from "@/components/PageHeader";
import { Button } from "@/components/ui/button";
import { WelcomeLaterRow } from "@/components/welcome/WelcomeLaterRow";
import { useAreaAccess } from "@/hooks/AccessHooks";
import { useFetchProjects } from "@/hooks/ProjectHooks";
import { useWelcomeFlow } from "@/hooks/useWelcomeFlow";
import { useSelectedWorkspace } from "@/hooks/WorkspaceHooks";
import { laterTitle } from "@/models/WelcomeWizard";
import { workspacePath } from "@/models/Workspace";
import { cn } from "@/lib/utils";

interface Exit {
  label: string;
  to: string;
}

// A workspace with no project yet, for someone who may create one, is empty until they do, so the way out leads there.
const useExit = (): Exit => {
  const workspace = useSelectedWorkspace();
  const { data: projects } = useFetchProjects();
  const can = useAreaAccess();
  if (!workspace) return { label: "Go to Nexul", to: "/" };
  if (projects?.length === 0 && can?.("newProject")) {
    return { label: "Create your first project", to: workspacePath(workspace.slug, "/wizard/project/project") };
  }
  return { label: `Go to ${workspace.name}`, to: `/${workspace.slug}` };
};

// How the wizard ends: each step not done as a row with Open, and one way into the app.
export const WelcomeFinish = () => {
  const { steps, marks, leave, leaving } = useWelcomeFlow();
  const exit = useExit();
  const later = steps.filter((step) => marks[step] !== "done");
  return (
    <>
      <h1 className={cn(displayTitleClass, "text-[2rem]")}>{laterTitle(later.length)}</h1>
      {later.length === 0 && <p className="mt-2 text-muted-foreground">Everything on your list is done.</p>}
      {later.length > 0 && (
        <p className="mt-2 text-muted-foreground">
          They wait under Welcome in your account menu. Only the ones still to do show a red dot there.
        </p>
      )}
      {later.length > 0 && (
        <ul className="mt-8 divide-y divide-border border-y border-border">
          {later.map((step) => (
            <WelcomeLaterRow key={step} step={step} />
          ))}
        </ul>
      )}
      <div className="flex justify-end pt-8">
        <Button type="button" loading={leaving} onClick={() => leave(exit.to)}>
          {exit.label}
        </Button>
      </div>
    </>
  );
};
