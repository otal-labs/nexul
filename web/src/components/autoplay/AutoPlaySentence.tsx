import type { ReactNode } from "react";

import { autoPlayLead, ruleText, type AutoPlayGroup, type AutoPlayMatch, type AutoPlayRule, type AutoPlaySubject } from "@nexul/client-core/autoPlayText";

import { useAutoPlayValues } from "@/hooks/AutoPlayValueHooks";
import type { AutoPlay } from "@/models/AutoPlay";

const lowerFirst = (text: string) => text.charAt(0).toLowerCase() + text.slice(1);

// One rule in words after the moment, "developer is Sam"; each reads its own field's values.
const RuleWords = ({ rule }: { rule: AutoPlayRule }) => {
  const values = useAutoPlayValues(rule.field);
  return <>{lowerFirst(ruleText(rule, values.label))}</>;
};

const joined = (parts: ReactNode[], match: AutoPlayMatch) =>
  parts.flatMap((part, i) => (i === 0 ? [part] : [<span key={`j${i}`}> {match === "all" ? "and" : "or"} </span>, part]));

const GroupWords = ({ group, bracket }: { group: AutoPlayGroup; bracket: boolean }) => (
  <>
    {bracket && group.rules.length > 1 && "("}
    {joined(group.rules.map((rule, i) => <RuleWords key={i} rule={rule} />), group.match)}
    {bracket && group.rules.length > 1 && ")"}
  </>
);

interface AutoPlaySentenceProps {
  autoPlay: AutoPlay;
  subject: AutoPlaySubject;
}

// An auto play as one sentence: "When it enters In progress, developer is Sam".
export const AutoPlaySentence = ({ autoPlay, subject }: AutoPlaySentenceProps) => {
  const groups = autoPlay.conditions.groups.filter((group) => group.rules.length > 0);
  return (
    <>
      {autoPlayLead(autoPlay, subject)}
      {groups.length > 0 && ", "}
      {joined(groups.map((group, i) => <GroupWords key={i} group={group} bracket={groups.length > 1} />), autoPlay.conditions.match)}
    </>
  );
};
