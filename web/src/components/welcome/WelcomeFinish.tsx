import { displayTitleClass } from "@/components/PageHeader";
import { Button } from "@/components/ui/button";
import { WelcomeLaterRow } from "@/components/welcome/WelcomeLaterRow";
import { useWelcomeFlow } from "@/hooks/useWelcomeFlow";
import { useSelectedWorkspace } from "@/hooks/WorkspaceHooks";
import { laterTitle } from "@/models/WelcomeWizard";
import { cn } from "@/lib/utils";

// How the wizard ends: each step not done as a row with Open, and one way into the app.
export const WelcomeFinish = () => {
  const { steps, marks, leave, leaving } = useWelcomeFlow();
  const workspace = useSelectedWorkspace();
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
        <Button type="button" loading={leaving} onClick={() => leave(workspace ? `/${workspace.slug}` : "/")}>
          {workspace ? `Go to ${workspace.name}` : "Go to Nexul"}
        </Button>
      </div>
    </>
  );
};
