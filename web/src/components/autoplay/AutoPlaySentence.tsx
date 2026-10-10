import type { ReactNode } from "react";

import { useAutoPlayValues } from "@/hooks/AutoPlayValueHooks";
import {
  FIELD_LABELS,
  LEVEL_LABELS,
  MOMENT_PHRASES,
  opLabel,
  RUN_ON_LABELS,
  type AutoPlay,
  type AutoPlaySubject,
} from "@/models/AutoPlay";
import { PLAY_STAGE_LABELS } from "@/models/Play";

const Part = ({ children }: { children: ReactNode }) => <span className="font-medium text-foreground">{children}</span>;

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
  const stage = autoPlay.moment_stage ? ` ${PLAY_STAGE_LABELS[autoPlay.moment_stage]}` : "";
  const condition = first && [FIELD_LABELS[first.field], opLabel(first.field, first.op), (first.values ?? []).map(values.label).join(" or ")]
    .filter(Boolean)
    .join(" ");

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
      {rules.length > 1 && (
        <>
          {" "}
          {autoPlay.conditions.match === "all" ? "and" : "or"} <Part>{rules.length - 1} more</Part>
        </>
      )}{" "}
      → <Part>{LEVEL_LABELS[top?.level ?? autoPlay.priority.otherwise]}</Part>
      {top && (
        <>
          , else <Part>{LEVEL_LABELS[autoPlay.priority.otherwise]}</Part>
        </>
      )}
      , runs on <Part>{RUN_ON_LABELS[autoPlay.run_on]}</Part>
    </p>
  );
};
