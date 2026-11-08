import type { ReactNode } from "react";

import { personLabel } from "@nexul/client-core/person";

import { useAutoPlayValues } from "@/hooks/AutoPlayValueHooks";
import { useFetchMe } from "@/hooks/AuthHooks";
import { useComputerNames } from "@/hooks/ComputerHooks";
import { usePersonLookup } from "@/hooks/PeopleHooks";
import {
  FIELD_LABELS,
  LEVEL_LABELS,
  MOMENT_PHRASES,
  opLabel,
  PRESENCE_PHRASES,
  RUN_ON_PHRASES,
  storedComputerIds,
  type AutoPlay,
  type AutoPlayGroup,
  type AutoPlayMatch,
  type AutoPlayRule,
  type AutoPlaySubject,
} from "@/models/AutoPlay";
import { PLAY_STAGE_LABELS } from "@/models/Play";
import { describeSchedule, DEFAULT_SWEEP_RUNS } from "@/models/Schedule";
import { cn } from "@/lib/utils";

// A value: foreground on a live rule, dimmed on one switched off (the row's data-off) but still above the muted labels and glue.
const Part = ({ children, className }: { children: ReactNode; className?: string }) => (
  <span className={cn("text-foreground group-data-off/row:text-foreground/80", className)}>{children}</span>
);

// A value list reads "frontend, backend or infrastructure", so its own "or" is never confused with the one between rules.
const listed = (values: string[]) => (values.length < 2 ? values.join("") : `${values.slice(0, -1).join(", ")} or ${values.at(-1)}`);

// The computers a chosen-computer auto play tries, in order, named as the viewer reads them; its person's own order when it names none.
const ComputersPhrase = ({ autoPlay }: { autoPlay: AutoPlay }) => {
  const ids = storedComputerIds(autoPlay);
  const names = useComputerNames(ids);
  const person = usePersonLookup(autoPlay.workspace_id);
  const { data: me } = useFetchMe();
  const yours = me?.user.id === autoPlay.run_as;
  const owner = yours ? "Your" : `${personLabel(person(autoPlay.run_as))}'s`;

  if (autoPlay.run_as === "") return "A computer nobody has picked yet";
  if (ids.length === 0) return `${owner} computers in ${yours ? "your" : "their"} order`;
  return names.map((name, i) => name || `${i === 0 ? "A" : "a"} computer you can't see`).join(", then ");
};

// One rule in words, "Type is Bug or Incident"; each reads its own field's values. A short rule never breaks across lines.
const RuleText = ({ rule }: { rule: AutoPlayRule }) => {
  const values = useAutoPlayValues(rule.field);
  const presence = (rule.op === "set" || rule.op === "unset") && PRESENCE_PHRASES[rule.field]?.[rule.op];
  const text = presence || [FIELD_LABELS[rule.field], opLabel(rule.field, rule.op), listed((rule.values ?? []).map(values.label))].filter(Boolean).join(" ");
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

// The moment that starts it: "enters In progress".
const momentPhrase = (autoPlay: AutoPlay) =>
  `${MOMENT_PHRASES[autoPlay.moment]}${autoPlay.moment_stage ? ` ${PLAY_STAGE_LABELS[autoPlay.moment_stage]}` : ""}`;

// A schedule's title is the schedule in words; every other moment's is "When a ticket becomes unblocked".
const titleOf = (autoPlay: AutoPlay, subject: AutoPlaySubject) =>
  autoPlay.moment === "schedule" && autoPlay.schedule
    ? describeSchedule(autoPlay.schedule.cron, autoPlay.schedule.timezone ?? "UTC")
    : `When a ${subject} ${momentPhrase(autoPlay)}`;

// How much one sweep takes: "Up to 10 tickets, nothing starts after 07:00".
const sweepPhrase = (autoPlay: AutoPlay) => {
  const runs = autoPlay.schedule?.max_runs ?? DEFAULT_SWEEP_RUNS;
  const stop = autoPlay.schedule?.stop_at;
  return `Up to ${runs} ticket${runs === 1 ? "" : "s"}${stop ? `, nothing starts after ${stop}` : ""}`;
};

// The moment as the title, then If, Priority and Runs on as labelled lines, every condition and priority rule in full.
export const AutoPlayClauses = ({ autoPlay, subject }: AutoPlayClausesProps) => {
  const groups = autoPlay.conditions.groups.filter((group) => group.rules.length > 0);
  const priority = autoPlay.priority.rules.filter((rule) => rule.when.rules.length > 0);

  return (
    <span className="flex min-w-0 flex-col gap-1 [overflow-wrap:anywhere]">
      <span className="text-sm font-medium text-foreground group-data-off/row:text-muted-foreground">
        {titleOf(autoPlay, subject)}
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
            {autoPlay.run_on !== "computer" && RUN_ON_PHRASES[autoPlay.run_on]}
            {autoPlay.run_on === "computer" && <ComputersPhrase autoPlay={autoPlay} />}
          </Part>
        </Clause>
      </span>
    </span>
  );
};
