import { ChevronRight, FileText, LayoutDashboard, UserPlus, type LucideIcon } from "lucide-react";
import type { ReactNode } from "react";

import { Button } from "@/components/ui/button";
import { WizardFooter } from "@/components/wizard/WizardFooter";
import { WizardInterviewOffer } from "@/components/wizard/WizardInterviewOffer";
import { wizardStepLabel, type WizardStepId } from "@/models/ProjectWizard";

import { PICKED, PROJECT_NAME } from "./mock";

export type Exit = "none" | "attached";

interface DoneProps {
  exit: Exit;
  onBack: () => void;
}

const Headline = ({ exit }: { exit: Exit }) => (
  <p className="text-sm font-medium">
    {exit === "none" && <>{PROJECT_NAME} is ready for docs and tickets.</>}
    {exit === "attached" && (
      <>
        <span className="font-mono">{PICKED.full_name}</span> is attached; deploy it whenever it's ready.
      </>
    )}
  </p>
);

const LeftRow = ({ label, state, action }: { label: string; state: string; action: string }) => (
  <li className="flex items-center gap-3 py-2">
    <span className="flex-1 text-sm">{label}</span>
    <span className="font-mono text-xs text-muted-foreground">{state}</span>
    <Button type="button" variant="ghost" size="sm">
      {action}
    </Button>
  </li>
);

const LeftForLater = ({ children }: { children: ReactNode }) => (
  <section aria-labelledby="pp-left" className="space-y-1">
    <h3 id="pp-left" className="text-sm font-medium">
      Left for later
    </h3>
    <ul className="divide-y divide-border border-y border-border">{children}</ul>
  </section>
);

const offer = <WizardInterviewOffer projectName={PROJECT_NAME} onStart={() => undefined} onSkip={() => undefined} />;

// Variant 1: ADR 0143's Done as it stands: every deploy step listed as skipped, the onward links in the footer.
export const DonePerStep = ({ exit, onBack }: DoneProps) => {
  const skipped: WizardStepId[] = exit === "none" ? ["repository", "service", "reach", "branches"] : ["service", "reach", "branches"];
  return (
    <div className="space-y-5">
      <div>
        <Headline exit={exit} />
        <p className="mt-1 text-sm text-muted-foreground">
          {exit === "none" && "The deploy steps wait for a repository. Open Repository any time to add one."}
          {exit === "attached" && "Its deploy steps are skipped for now; the project's Services offers them again."}
        </p>
      </div>
      <LeftForLater>
        {skipped.map((id) => (
          <LeftRow key={id} label={wizardStepLabel(id, false)} state="skipped" action="Open" />
        ))}
      </LeftForLater>
      {offer}
      <WizardFooter onBack={onBack}>
        <Button variant="outline">Docs</Button>
        <Button variant="outline">Invite people</Button>
        <Button variant="outline">Finish</Button>
      </WizardFooter>
    </div>
  );
};

const NextRow = ({ icon: Icon, label, why }: { icon: LucideIcon; label: string; why: string }) => (
  <li>
    <button
      type="button"
      className="flex w-full items-center gap-3 px-2 py-3 text-left transition-colors duration-150 ease-standard hover:bg-accent/40"
    >
      <Icon className="size-4 shrink-0 text-muted-foreground" aria-hidden />
      <span className="min-w-0 flex-1">
        <span className="block text-sm font-medium">{label}</span>
        <span className="block truncate text-sm text-muted-foreground">{why}</span>
      </span>
      <ChevronRight className="size-4 shrink-0 text-muted-foreground" aria-hidden />
    </button>
  </li>
);

// Variant 2: the deploy steps fold into one row, and the onward places are rows with a reason each.
export const DoneNextPlaces = ({ exit, onBack }: DoneProps) => (
  <div className="space-y-6">
    <Headline exit={exit} />
    <LeftForLater>
      {exit === "none" && <LeftRow label="Deploying" state="needs a repository" action="Open Repository" />}
      {exit === "attached" && <LeftRow label="Deploying" state="not started" action="Deploy it" />}
    </LeftForLater>
    {offer}
    <section aria-labelledby="pp-next" className="space-y-1">
      <h3 id="pp-next" className="text-sm font-medium">
        Where to next
      </h3>
      <ul className="divide-y divide-border border-y border-border">
        <NextRow icon={FileText} label="Docs" why="Write the brief everyone reads first." />
        <NextRow icon={LayoutDashboard} label="Board" why="Add the first tickets." />
        <NextRow icon={UserPlus} label="Invite people" why="Bring in the client and the team." />
      </ul>
    </section>
    <WizardFooter onBack={onBack}>
      <Button variant="outline">Finish</Button>
    </WizardFooter>
  </div>
);
