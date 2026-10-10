import type { ReactNode } from "react";

import { useAutoPlayValues, type FieldValues } from "@/hooks/AutoPlayValueHooks";
import {
  FIELD_LABELS,
  LEVEL_LABELS,
  MOMENT_PHRASES,
  opLabel,
  RUN_ON_LABELS,
  type AutoPlay,
  type AutoPlayGroup,
  type AutoPlayRule,
  type AutoPlaySubject,
} from "@/models/AutoPlay";
import { PLAY_STAGE_LABELS } from "@/models/Play";

const Part = ({ children }: { children: ReactNode }) => <span className="font-medium text-foreground">{children}</span>;

const ruleText = (rule: AutoPlayRule | undefined, values: FieldValues) =>
  rule && [FIELD_LABELS[rule.field], opLabel(rule.field, rule.op), (rule.values ?? []).map(values.label).join(" or ")]
    .filter(Boolean)
    .join(" ");

// How many rules follow the one spelled out, joined the way their group matches.
const More = ({ count, match }: { count: number; match: AutoPlayGroup["match"] }) =>
  count > 0 && (
    <>
      {" "}
      {match === "all" ? "and" : "or"} <Part>{count} more</Part>
    </>
  );

interface AutoPlaySentenceProps {
  autoPlay: AutoPlay;
  subject: AutoPlaySubject;
}

// The variable parts in foreground weight and the glue words muted, so the row reads as one sentence.
export const AutoPlaySentence = ({ autoPlay, subject }: AutoPlaySentenceProps) => {
  const rules = autoPlay.conditions.groups.flatMap((group) => group.rules);
  const [first] = rules;
  const values = useAutoPlayValues(first?.field ?? "project");
  const top = autoPlay.priority.rules[0];
  const [topFirst] = top?.when.rules ?? [];
  const topValues = useAutoPlayValues(topFirst?.field ?? "project");
  const stage = autoPlay.moment_stage ? ` ${PLAY_STAGE_LABELS[autoPlay.moment_stage]}` : "";
  const condition = ruleText(first, values);
  const topCondition = ruleText(topFirst, topValues);

  return (
    <p className="text-sm text-muted-foreground [overflow-wrap:anywhere]">
      When a {subject}{" "}
      <Part>
        {MOMENT_PHRASES[autoPlay.moment]}
        {stage}
      </Part>
      {condition && (
        <>
          , if <Part>{condition}</Part>
        </>
      )}
      <More count={rules.length - 1} match={autoPlay.conditions.match} />{" "}
      → <Part>{LEVEL_LABELS[top?.level ?? autoPlay.priority.otherwise]}</Part>
      {top && topCondition && (
        <>
          {" "}
          if <Part>{topCondition}</Part>
          <More count={top.when.rules.length - 1} match={top.when.match} />
        </>
      )}
      {top && (
        <>
          , else <Part>{LEVEL_LABELS[autoPlay.priority.otherwise]}</Part>
        </>
      )}
      , runs on <Part>{RUN_ON_LABELS[autoPlay.run_on]}</Part>
    </p>
  );
};
