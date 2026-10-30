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
  RUN_ON_LABELS,
  storedComputerIds,
  type AutoPlay,
  type AutoPlayGroup,
  type AutoPlayMatch,
  type AutoPlayRule,
  type AutoPlaySubject,
} from "@/models/AutoPlay";
import { PLAY_STAGE_LABELS } from "@/models/Play";
import { cn } from "@/lib/utils";

const Part = ({ children, className }: { children: ReactNode; className?: string }) => <span className={cn("font-medium text-foreground", className)}>{children}</span>;

// A value list reads "frontend, backend or infrastructure", so its own "or" is never confused with the one between rules.
const listed = (values: string[]) => (values.length < 2 ? values.join("") : `${values.slice(0, -1).join(", ")} or ${values.at(-1)}`);

// The computers a chosen-computer auto play tries, in order, named as the viewer reads them; its person's own order when it names none.
const ComputersPhrase = ({ autoPlay }: { autoPlay: AutoPlay }) => {
  const ids = storedComputerIds(autoPlay);
  const names = useComputerNames(ids);
  const person = usePersonLookup(autoPlay.workspace_id);
  const { data: me } = useFetchMe();
  const yours = me?.user.id === autoPlay.run_as;
  const owner = yours ? "your" : `${personLabel(person(autoPlay.run_as))}'s`;

  if (autoPlay.run_as === "") return "a computer nobody has picked yet";
  if (ids.length === 0) return `${owner} computers in ${yours ? "your" : "their"} order`;
  return names.map((name) => name || "a computer you can't see").join(", then ");
};

// One rule in words, "Type is Bug or Incident"; each reads its own field's values. A short rule never breaks across lines.
const RuleText = ({ rule }: { rule: AutoPlayRule }) => {
  const values = useAutoPlayValues(rule.field);
  const text = [FIELD_LABELS[rule.field], opLabel(rule.field, rule.op), listed((rule.values ?? []).map(values.label))].filter(Boolean).join(" ");
  return <Part className={cn(text.length <= 32 && "whitespace-nowrap")}>{text}</Part>;
};

const joined = (parts: ReactNode[], match: AutoPlayMatch) =>
  parts.flatMap((part, i) => (i === 0 ? [part] : [<span key={`j${i}`}>{match === "all" ? " and " : " or "}</span>, part]));

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

interface AutoPlaySentenceProps {
  autoPlay: AutoPlay;
  subject: AutoPlaySubject;
}

// The moment that starts it: "enters In progress".
const momentPhrase = (autoPlay: AutoPlay) =>
  `${MOMENT_PHRASES[autoPlay.moment]}${autoPlay.moment_stage ? ` ${PLAY_STAGE_LABELS[autoPlay.moment_stage]}` : ""}`;

// The moment as the title, then every condition and every priority rule in full; values in foreground weight, the glue muted.
export const AutoPlaySentence = ({ autoPlay, subject }: AutoPlaySentenceProps) => {
  const groups = autoPlay.conditions.groups.filter((group) => group.rules.length > 0);
  const priority = autoPlay.priority.rules.filter((rule) => rule.when.rules.length > 0);

  return (
    <span className="flex min-w-0 flex-col gap-0.5 [overflow-wrap:anywhere]">
      <span className="text-sm font-medium text-foreground">
        When a {subject} {momentPhrase(autoPlay)}
      </span>
      {groups.length > 0 && (
        <span className="text-xs text-pretty text-muted-foreground">
          {" "}if {joined(groups.map((group, i) => <GroupText key={i} group={group} bracket={groups.length > 1} />), autoPlay.conditions.match)}
        </span>
      )}
      <span className="text-xs text-pretty text-muted-foreground">
        {" "}
        <span aria-hidden className="relative -top-0.5">
          →
        </span>{" "}
        {priority.map((rule, i) => (
          <span key={i}>
            <Part>{LEVEL_LABELS[rule.level]}</Part>{"\u00a0if\u00a0"}<GroupText group={rule.when} bracket={false} />,{" "}
          </span>
        ))}
        {priority.length > 0 && "else\u00a0"}
        <Part>{LEVEL_LABELS[autoPlay.priority.otherwise]}</Part>,{" "}
        <span className={cn(autoPlay.run_on !== "computer" && "whitespace-nowrap")}>
          runs on{" "}
          <Part>
            {autoPlay.run_on !== "computer" && RUN_ON_LABELS[autoPlay.run_on]}
            {autoPlay.run_on === "computer" && <ComputersPhrase autoPlay={autoPlay} />}
          </Part>
        </span>
      </span>
    </span>
  );
};
