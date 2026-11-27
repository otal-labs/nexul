import { personLabel } from "@nexul/client-core/person";
import type { AutoPlayPlaces } from "@nexul/client-core/autoPlayComputers";

import { statusDotClass } from "@/components/settings/SettingsStatus";
import { usePerson } from "@/hooks/PeopleHooks";
import { cn } from "@/lib/utils";
import { isStaleOffline, offlineForDays } from "@/models/ComputerChecks";
import type { Computer } from "@/models/Pairing";

const pillClass = "inline-flex max-w-full items-center gap-1.5 rounded-md border border-border px-1.5 py-0.5 text-xs leading-4";

const ROLE_TEXT = { causer: "The mover's computers", developer: "The developer's computers", tester: "The tester's computers" } as const;

const PersonPill = ({ userId }: { userId: string }) => {
  const person = usePerson(userId);
  return <span className={cn(pillClass, "text-muted-foreground")}>{personLabel(person)}'s computers</span>;
};

interface ComputerPillProps {
  computer: Computer;
  struck?: boolean;
}

const ComputerPill = ({ computer, struck }: ComputerPillProps) => {
  const online = !!computer.runner?.connected;
  const days = offlineForDays(computer);
  const stale = isStaleOffline(computer);
  const tip = stale ? `Offline · seen ${days} ${days === 1 ? "day" : "days"} ago` : "Offline · passed over while offline";
  return (
    <span title={online || struck ? undefined : tip} className={cn(pillClass, "font-mono", (struck || !online) && "text-muted-foreground", struck && "line-through")}>
      {!struck && online && <span role="img" aria-label="Online" className={cn("size-1.5 shrink-0 rounded-full", statusDotClass.success)} />}
      {!struck && !online && (
        <span role="img" aria-label="Offline" className={cn("size-1.5 shrink-0 rounded-full border", stale ? "border-warning-mark" : "border-muted-foreground")} />
      )}
      <span className="truncate">{computer.name}</span>
      {stale && <span aria-hidden className="shrink-0 text-muted-foreground">· {days}d</span>}
      {!struck && !online && <span className="sr-only">, offline{stale ? ` ${days} ${days === 1 ? "day" : "days"}` : ""}</span>}
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
    {places.kind === "role" && places.role !== "nobody" && <span className={cn(pillClass, "text-muted-foreground")}>{ROLE_TEXT[places.role]}</span>}
    {places.kind === "role" && places.role === "nobody" && (
      <span className={cn(pillClass, "text-muted-foreground")}>
        <span aria-hidden className={cn("size-1.5 shrink-0 rounded-full", statusDotClass.warning)} />
        Needs a person
      </span>
    )}
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
