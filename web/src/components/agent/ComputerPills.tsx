import { personLabel } from "@nexul/client-core/person";
import type { AutoPlayPlaces } from "@nexul/client-core/autoPlayComputers";

import { statusDotClass } from "@/components/settings/SettingsStatus";
import { usePerson } from "@/hooks/PeopleHooks";
import { cn } from "@/lib/utils";
import { runnerPresence } from "@/models/ComputerChecks";
import type { Computer } from "@/models/Pairing";

const pillClass = "inline-flex max-w-full items-center gap-1.5 rounded-md border border-border px-1.5 py-0.5 text-xs leading-4";

const ROLE_TEXT = { causer: "Whoever moved it: their computers", developer: "The developer: their computers", tester: "The tester: their computers", nobody: "Nobody picked yet" } as const;

const PersonPill = ({ userId }: { userId: string }) => {
  const person = usePerson(userId);
  return <span className={cn(pillClass, "text-muted-foreground")}>{personLabel(person)}: their computers</span>;
};

interface ComputerPillProps {
  computer: Computer;
  struck?: boolean;
}

const ComputerPill = ({ computer, struck }: ComputerPillProps) => {
  const state = runnerPresence(computer.runner);
  return (
    <span className={cn(pillClass, "font-mono", struck && "text-muted-foreground line-through")}>
      {!struck && <span role="img" aria-label={state.text} className={cn("size-1.5 shrink-0 rounded-full", statusDotClass[state.tone])} />}
      <span className="truncate">{computer.name}</span>
    </span>
  );
};

interface ComputerPillsProps {
  places: AutoPlayPlaces<Computer>;
  // The editor lists the computers the rule leaves out too, struck through; a row names only where it would run.
  showExcluded?: boolean;
}

// Each computer an auto play would run on, with its state dot, or one muted pill saying whose computers it is.
export const ComputerPills = ({ places, showExcluded = false }: ComputerPillsProps) => (
  <>
    {places.kind === "role" && <span className={cn(pillClass, "text-muted-foreground")}>{ROLE_TEXT[places.role]}</span>}
    {places.kind === "person" && <PersonPill userId={places.userId} />}
    {places.kind === "computers" &&
      places.computers
        .filter((reach) => reach.reached || showExcluded)
        .map((reach) => <ComputerPill key={reach.computer.id} computer={reach.computer} struck={!reach.reached} />)}
    {places.kind === "computers" && !showExcluded && places.computers.every((reach) => !reach.reached) && (
      <span className={cn(pillClass, "text-muted-foreground")}>No computer takes it</span>
    )}
  </>
);
