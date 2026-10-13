import type { ReactNode } from "react";
import { Check, ChevronRight, Link2Off } from "lucide-react";

import { Microheader } from "@/components/Microheader";
import { SettingsStatus } from "@/components/settings/SettingsStatus";
import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { avatarGradient } from "@/lib/avatarGradient";
import { cn, initials } from "@/lib/utils";

import { doneDetail, leftDetail, people, stepName, tally, type ProtoPerson, type ProtoStep } from "./data";

export const Avatar = ({ person, className }: { person: ProtoPerson; className?: string }) => (
  <span
    aria-hidden
    style={{ backgroundImage: avatarGradient(person.login) }}
    className={cn("@container flex shrink-0 items-center justify-center rounded-full font-semibold text-white select-none", className)}
  >
    <span className="text-[max(10px,38cqw)] leading-none [text-shadow:0_1px_2px_oklch(0_0_0/0.35)]">{initials(person.name)}</span>
  </span>
);

interface RowShellProps {
  person: ProtoPerson;
  onOpen: (id: string) => void;
  extraLine?: ReactNode;
  column?: ReactNode;
}

// The Team row as it ships today, with two seams: a line under the meta line, and a column before presence.
export const RowShell = ({ person, onOpen, extraLine, column }: RowShellProps) => (
  <li>
    <button
      type="button"
      onClick={() => onOpen(person.id)}
      aria-label={`Open ${person.name}`}
      className="flex w-full items-center gap-3 px-3 py-3 text-left transition-colors duration-150 ease-standard hover:bg-accent/40 sm:px-4"
    >
      <span className="relative shrink-0 self-start pt-0.5">
        <Avatar person={person} className="size-8" />
        <span
          aria-hidden
          className={cn(
            "absolute -right-0.5 -bottom-0.5 size-2.5 rounded-full ring-2 ring-card",
            person.online ? "bg-success" : "bg-muted-foreground",
          )}
        />
      </span>
      <span className="min-w-0 flex-1">
        <span dir="auto" className="line-clamp-2 font-medium break-words" title={person.name}>
          {person.name}
        </span>
        <span className="block truncate text-xs text-muted-foreground">{person.membership}</span>
        {extraLine}
      </span>
      {column}
      <span className="flex shrink-0 flex-col items-end gap-1 self-start text-right text-sm text-muted-foreground">
        <span className={cn(person.online && "text-foreground")}>{person.presence}</span>
        {person.status === "disabled" && <SettingsStatus tone="warning">Disabled</SettingsStatus>}
      </span>
      <ChevronRight aria-hidden className="size-4 shrink-0 self-center text-muted-foreground" />
    </button>
  </li>
);

// An account that can't sign in has nothing to set up, so its progress is not shown.
export const showsProgress = (person: ProtoPerson) => person.status === "active";

const StepRow = ({ step }: { step: ProtoStep }) => (
  <li className="flex items-start gap-3 px-3 py-2.5 sm:px-4">
    <div className="min-w-0 flex-1">
      <p className={cn("text-sm font-medium", step.state !== "left" && "text-muted-foreground")}>{stepName[step.key]}</p>
      <p className="text-xs text-muted-foreground">
        {step.state === "left" && leftDetail[step.key]}
        {step.state === "done" && doneDetail[step.key]}
        {step.state === "skipped" && "They chose to skip it; it comes back if a new role gives it"}
      </p>
    </div>
    <span className="shrink-0 pt-0.5">
      {step.state === "done" && <SettingsStatus tone="success">Done</SettingsStatus>}
      {step.state === "left" && <SettingsStatus tone="warning">Not done</SettingsStatus>}
      {step.state === "skipped" && (
        <SettingsStatus tone="muted" detail={step.at}>
          Skipped
        </SettingsStatus>
      )}
    </span>
  </li>
);

// The person's Welcome wizard: what is left or skipped as rows, what is done folded into one line under them.
export const WelcomeBlock = ({ person }: { person: ProtoPerson }) => {
  const { done, total } = tally(person.steps);
  const open = person.steps.filter((step) => step.state !== "done");
  const finished = person.steps.filter((step) => step.state === "done");
  return (
    <section aria-labelledby="welcome-title" className="space-y-2 px-6 pb-4">
      <div className="flex items-baseline justify-between gap-3">
        <Microheader id="welcome-title">Welcome wizard</Microheader>
        <span className="font-mono text-xs text-muted-foreground tabular-nums">
          {done} of {total} set up
        </span>
      </div>
      {open.length > 0 && (
        <ul className="divide-y divide-border overflow-hidden rounded-md border">
          {open.map((step) => (
            <StepRow key={step.key} step={step} />
          ))}
        </ul>
      )}
      {finished.length > 0 && (
        <p className="flex items-center gap-1.5 text-xs text-muted-foreground">
          <Check aria-hidden className="size-3.5 text-success" />
          {open.length === 0 && <span>Every step done: {finished.map((step) => stepName[step.key]).join(", ")}</span>}
          {open.length > 0 && <span>Done: {finished.map((step) => stepName[step.key]).join(", ")}</span>}
        </p>
      )}
    </section>
  );
};

// The person dialog, cut down to its header, the Welcome block, and a stub of the workspace tabs it already has.
export const PersonDialog = ({ personId, onClose }: { personId: string | null; onClose: () => void }) => {
  const person = people.find((candidate) => candidate.id === personId);
  return (
    <Dialog open={!!person} onOpenChange={(open) => !open && onClose()}>
      <DialogContent className="flex max-h-[min(90dvh,52rem)] flex-col gap-0 overflow-y-auto p-0 sm:max-w-[40rem]">
        {person && (
          <DialogHeader className="flex-row items-center gap-3 px-6 pt-6 pb-4 pr-12 text-left">
            <Avatar person={person} className="size-10" />
            <div className="min-w-0 flex-1 space-y-1">
              <DialogTitle dir="auto" className="line-clamp-2 break-words">
                {person.name}
              </DialogTitle>
              <DialogDescription className="truncate font-mono text-xs">@{person.login}</DialogDescription>
              <p className="text-sm text-muted-foreground">
                {person.presence}
                {person.joined && (
                  <span>
                    {" "}
                    · Joined {person.joined.at}, invited by {person.joined.by}
                  </span>
                )}
              </p>
            </div>
          </DialogHeader>
        )}
        {person && showsProgress(person) && <WelcomeBlock person={person} />}
        {person && (
          <div className="border-t border-border px-6 pt-3 pb-6">
            <div className="flex gap-1 border-b border-border text-sm">
              <span className="rounded-t-md border border-b-0 border-border px-3 py-1.5">Acme</span>
              <span className="px-3 py-1.5 text-muted-foreground">Globex</span>
            </div>
            <p className="pt-4 text-sm text-muted-foreground">Role, overrides and Project access, as today.</p>
          </div>
        )}
      </DialogContent>
    </Dialog>
  );
};

const AcceptedRow = ({ person, onOpen, progress }: { person: ProtoPerson; onOpen: (id: string) => void; progress: ReactNode }) => (
  <li>
    <button
      type="button"
      onClick={() => onOpen(person.id)}
      className="flex w-full items-center gap-3 px-3 py-3 text-left transition-colors duration-150 ease-standard hover:bg-accent/40 sm:px-4"
    >
      <Avatar person={person} className="size-7" />
      <span className="min-w-0 flex-1">
        <span dir="auto" className="block truncate text-sm font-medium" title={person.name}>{person.name}</span>
        <span className="block truncate text-xs text-muted-foreground">
          {person.joined?.grant} · invited by {person.joined?.by}
        </span>
        {progress}
      </span>
      <span className="shrink-0 font-mono text-xs text-muted-foreground tabular-nums">accepted {person.joined?.at}</span>
      <ChevronRight aria-hidden className="size-4 shrink-0 text-muted-foreground" />
    </button>
  </li>
);

export const InvitationsTab = ({ onOpen, renderProgress }: { onOpen: (id: string) => void; renderProgress: (person: ProtoPerson) => ReactNode }) => {
  const accepted = ["maxi", "sam", "bob", "lena"].map((id) => people.find((person) => person.id === id)!);
  return (
    <div className="space-y-6">
      <section className="space-y-3">
        <div className="space-y-1">
          <Microheader>Active invitation links</Microheader>
          <p className="text-sm text-muted-foreground">Each link works once and is shown only when it's created.</p>
        </div>
        <ul className="divide-y divide-border overflow-hidden rounded-md border">
          <li className="flex items-center gap-3 px-3 py-3 sm:px-4">
            <span className="flex size-7 shrink-0 items-center justify-center rounded-md bg-muted text-muted-foreground">
              <Link2Off className="size-4" aria-hidden />
            </span>
            <div className="min-w-0 flex-1">
              <p className="truncate text-sm font-medium">Acme · Developer</p>
              <p className="truncate text-xs text-muted-foreground">by Alice Moreau · expires in 6 days</p>
            </div>
          </li>
        </ul>
      </section>
      <section className="space-y-3">
        <div className="space-y-1">
          <Microheader>Accepted in the last 30 days</Microheader>
          <p className="text-sm text-muted-foreground">Who each link let in, and how far their setup got.</p>
        </div>
        <ul className="divide-y divide-border overflow-hidden rounded-md border">
          {accepted.map((person) => (
            <AcceptedRow key={person.id} person={person} onOpen={onOpen} progress={renderProgress(person)} />
          ))}
        </ul>
      </section>
    </div>
  );
};
