import { personLabel } from "@nexul/client-core/person";
import type { AutoPlayPlaces } from "@nexul/client-core/autoPlayComputers";

import { formatShortAgo } from "@/components/doc/docTime";
import { statusDotClass } from "@/components/settings/SettingsStatus";
import { useFetchPresence } from "@/hooks/PairingHooks";
import { usePerson } from "@/hooks/PeopleHooks";
import { cn } from "@/lib/utils";
import { computerReadiness, offlineForDays, readinessDot } from "@/models/ComputerChecks";
import type { Computer } from "@/models/Pairing";

export const pillClass = "inline-flex max-w-full items-center gap-1.5 rounded-md border border-border px-1.5 py-0.5 text-xs leading-4";

const ROLE_TEXT = { causer: "The mover's computers", developer: "The developer's computers", tester: "The tester's computers" } as const;

const PersonPill = ({ userId }: { userId: string }) => {
  const person = usePerson(userId);
  return <span className={cn(pillClass, "text-muted-foreground")}>{personLabel(person)}'s computers</span>;
};

interface ComputerPillProps {
  computer: Computer;
  struck?: boolean;
}

const RING: Record<string, string> = { muted: "border-muted-foreground", warning: "border-warning-mark" };

const ComputerPill = ({ computer, struck }: ComputerPillProps) => {
  const presence = useFetchPresence();
  const { state, words } = computerReadiness(computer, presence.data?.[computer.id]);
  const dot = readinessDot(state);
  const days = offlineForDays(computer);
  const offline = state === "offline" || state === "stale";
  const age = state === "stale" ? `${days}d` : computer.runner?.last_seen ? formatShortAgo(computer.runner.last_seen) : "";
  const note = (() => {
    if (state === "degraded") return words;
    return offline && age ? `seen ${age}` : "";
  })();
  const tip = (() => {
    if (struck || state === "ready") return undefined;
    if (state === "stale") return `Offline · seen ${days} ${days === 1 ? "day" : "days"} ago`;
    if (state === "offline") return "Offline · passed over while offline";
    return words;
  })();
  return (
    <span title={tip} className={cn(pillClass, "font-mono", (struck || state !== "ready") && "text-muted-foreground", struck && "line-through")}>
      {!struck && !dot.hollow && <span role="img" aria-label={state === "ready" ? "Online" : "Not ready"} className={cn("size-1.5 shrink-0 rounded-full", statusDotClass[dot.tone])} />}
      {!struck && dot.hollow && <span role="img" aria-label="Offline" className={cn("size-1.5 shrink-0 rounded-full border", RING[dot.tone])} />}
      <span className="truncate">{computer.name}</span>
      {!struck && note && <span aria-hidden className="shrink-0 text-muted-foreground">· {note}</span>}
      {!struck && state === "degraded" && <span className="sr-only">, {words}</span>}
      {!struck && offline && <span className="sr-only">, offline{state === "stale" ? ` ${days} ${days === 1 ? "day" : "days"}` : ""}</span>}
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
    {places.kind === "role" && places.role === "nobody" && <span className={cn(pillClass, "border-dashed text-muted-foreground")}>Runs as not set</span>}
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
