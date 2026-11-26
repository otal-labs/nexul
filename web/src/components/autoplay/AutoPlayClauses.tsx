import type { ReactNode } from "react";

import { personLabel } from "@nexul/client-core/person";
import { autoPlayTitle, LEVEL_LABELS, runOnPhrase, ruleText, type AutoPlayGroup, type AutoPlayMatch, type AutoPlayRule, type AutoPlaySubject } from "@nexul/client-core/autoPlayText";

import { useAutoPlayValues } from "@/hooks/AutoPlayValueHooks";
import { useFetchMe } from "@/hooks/AuthHooks";
import { usePersonLookup } from "@/hooks/PeopleHooks";
import type { AutoPlay } from "@/models/AutoPlay";
import { DEFAULT_SWEEP_RUNS } from "@/models/Schedule";
import { cn } from "@/lib/utils";

// A value: foreground on a live rule, dimmed on one switched off (the row's data-off) but still above the muted labels and glue.
const Part = ({ children, className }: { children: ReactNode; className?: string }) => (
  <span className={cn("text-foreground group-data-off/row:text-foreground/80", className)}>{children}</span>
);

// Whose computers a person auto play runs on, "Your computers" or "Sam's computers"; nobody's until someone is picked.
const runOnOwner = (runAs: string, me: string | undefined, name: string) => {
  if (runAs === "") return "";
  return me === runAs ? "Your" : `${name}'s`;
};

// One rule in words, "Type is Bug or Incident"; each reads its own field's values. A short rule never breaks across lines.
const RuleText = ({ rule }: { rule: AutoPlayRule }) => {
  const values = useAutoPlayValues(rule.field);
  const text = ruleText(rule, values.label);
  return <Part className={cn(text.length <= 32 && "whitespace-nowrap")}>{text}</Part>;
};

// The joiner keeps to the rule after it, so a wrapped line always starts with its "and" or "or".
const joined = (parts: ReactNode[], match: AutoPlayMatch) =>
  parts.flatMap((part, i) => (i === 0 ? [part] : [<span key={`j${i}`}> {match === "all" ? "and" : "or"}{"\u00a0"}</span>, part]));

const GroupText = ({ group, bracket }: { group: AutoPlayGroup; bracket: boolean }) => {
  const rules = joined(group.rules.map((rule, i) => <RuleText key={i} rule={rule} />), group.match);
  return (
    <>
      {bracket && group.rules.length > 1 && "("}
      {rules}
      {bracket && group.rules.length > 1 && ")"}
    </>
  );
};

// One labelled clause: the label in the muted column, the value hanging beside it however far it wraps.
const Clause = ({ label, children }: { label: string; children: ReactNode }) => (
  <>
    <span>{label}</span> <span className="min-w-0 text-pretty">{children}</span>{" "}
  </>
);

interface AutoPlayClausesProps {
  autoPlay: AutoPlay;
  subject: AutoPlaySubject;
}

// How much one sweep takes: "Up to 10 tickets, nothing starts after 07:00".
const sweepPhrase = (autoPlay: AutoPlay) => {
  const runs = autoPlay.schedule?.max_runs ?? DEFAULT_SWEEP_RUNS;
  const stop = autoPlay.schedule?.stop_at;
  return `Up to ${runs} ticket${runs === 1 ? "" : "s"}${stop ? `, nothing starts after ${stop}` : ""}`;
};

// The moment as the title, then If, Priority and Runs on as labelled lines, every condition and priority rule in full.
export const AutoPlayClauses = ({ autoPlay, subject }: AutoPlayClausesProps) => {
  const { data: me } = useFetchMe();
  const person = usePersonLookup(autoPlay.workspace_id);
  const groups = autoPlay.conditions.groups.filter((group) => group.rules.length > 0);
  const priority = autoPlay.priority.rules.filter((rule) => rule.when.rules.length > 0);

  return (
    <span className="flex min-w-0 flex-col gap-1 [overflow-wrap:anywhere]">
      <span className="text-sm font-medium text-foreground group-data-off/row:text-muted-foreground">
        {autoPlayTitle(autoPlay, subject)}
      </span>{" "}
      <span className="grid grid-cols-[4.5rem_minmax(0,1fr)] gap-x-2 gap-y-1 text-xs text-muted-foreground">
        {groups.length > 0 && (
          <Clause label="If">{joined(groups.map((group, i) => <GroupText key={i} group={group} bracket={groups.length > 1} />), autoPlay.conditions.match)}</Clause>
        )}
        <Clause label="Priority">
          {priority.map((rule, i) => (
            <span key={i}>
              <Part>{LEVEL_LABELS[rule.level]}</Part>{" if "}<GroupText group={rule.when} bracket={false} />,{" "}
            </span>
          ))}
          {priority.length > 0 && "else "}
          <Part>{LEVEL_LABELS[autoPlay.priority.otherwise]}</Part>
        </Clause>
        {autoPlay.moment === "schedule" && (
          <Clause label="Sweep">
            <Part>{sweepPhrase(autoPlay)}</Part>
          </Clause>
        )}
        {autoPlay.one_at_a_time && (
          <Clause label="Limits">
            <Part>One run at a time</Part>
            {autoPlay.resource && (
              <>
                {" with "}
                <Part className="technical">{autoPlay.resource}</Part>
              </>
            )}
          </Clause>
        )}
        <Clause label="Runs on">
          <Part>
            {runOnPhrase(autoPlay.run_on, runOnOwner(autoPlay.run_as, me?.user.id, personLabel(person(autoPlay.run_as))))}
          </Part>
        </Clause>
      </span>
    </span>
  );
};
