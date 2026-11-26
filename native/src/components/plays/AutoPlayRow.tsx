import { View } from "react-native";

import { autoPlayTitle, FIELD_LABELS, normalRunOn, opLabel, runOnPhrase, ruleText, type AutoPlayRule } from "@nexul/client-core/autoPlayText";
import { personLabel } from "@nexul/client-core/person";
import { PLAY_STAGE_LABELS, type PlayStage } from "@nexul/client-core/plays";

import { ActionSwitch } from "@/components/action/ActionSwitch";
import { ErrorDisplay } from "@/components/ErrorDisplay";
import { Text } from "@/components/ui/text";
import { useSetAutoPlayEnabled } from "@/hooks/PlayHooks";
import { usePersonLookup } from "@/hooks/PeopleHooks";
import { useFetchProjects } from "@/hooks/ProjectHooks";
import { cn } from "@/lib/utils";
import { type AutoPlay } from "@/models/Play";

interface AutoPlayRowProps {
  autoPlay: AutoPlay;
  subject: "ticket" | "doc" | "stack";
  first: boolean;
}

// Fields whose values are ids the phone has no list to name; the web names them.
const UNNAMED = new Set(["status", "folder", "stack"]);

// "Status is one of 2 picked on the web" where the phone can't name the values.
const unnamedText = (rule: AutoPlayRule) => {
  const n = rule.values?.length ?? 0;
  return `${FIELD_LABELS[rule.field]} ${opLabel(rule.field, rule.op)} ${n === 1 ? "one" : `one of ${n}`} picked on the web`;
};

const useRuleLabel = (workspaceId: string) => {
  const { data: projects } = useFetchProjects();
  const person = usePersonLookup(workspaceId);
  return (rule: AutoPlayRule): string => {
    if (UNNAMED.has(rule.field) && (rule.op === "is" || rule.op === "is_not")) return unnamedText(rule);
    return ruleText(rule, (value) => {
      if (rule.field === "project") return projects?.find((p) => p.id === value)?.name ?? "a project you can't see";
      if (rule.field === "developer" || rule.field === "tester") return personLabel(person(value));
      if (rule.field === "stage") return PLAY_STAGE_LABELS[value as PlayStage] ?? value;
      return value;
    });
  };
};

// The auto play as the web's sentence: its moment, its conditions in words, whose computers it runs on; and its switch.
export const AutoPlayRow = ({ autoPlay, subject, first }: AutoPlayRowProps) => {
  const label = useRuleLabel(autoPlay.workspace_id);
  const update = useSetAutoPlayEnabled();
  const person = usePersonLookup(autoPlay.workspace_id);
  const isPerson = normalRunOn(autoPlay.run_on) === "person";
  const unpicked = isPerson && autoPlay.run_as === "";
  const groups = autoPlay.conditions.groups.filter((group) => group.rules.length > 0);
  const conditions = groups
    .map((group) => group.rules.map(label).join(group.match === "all" ? " and " : " or "))
    .map((text, _, all) => (all.length > 1 ? `(${text})` : text))
    .join(autoPlay.conditions.match === "all" ? " and " : " or ");
  const runsOn = runOnPhrase(autoPlay.run_on, autoPlay.run_as === "" ? "" : `${personLabel(person(autoPlay.run_as))}'s`);

  return (
    <View className={cn("gap-2 px-4 py-3", !first && "border-t border-border")}>
      <View className="flex-row items-start gap-3">
        <View className="min-w-0 flex-1 gap-1">
          <Text className={cn("text-body font-medium", !autoPlay.enabled && "text-muted-foreground")}>{autoPlayTitle(autoPlay, subject)}</Text>
          {conditions !== "" && <Text className="text-row text-muted-foreground">If {conditions}</Text>}
          <Text className="text-row text-muted-foreground">Runs on {isPerson ? runsOn : runsOn.charAt(0).toLowerCase() + runsOn.slice(1)}</Text>
        </View>
        <ActionSwitch
          permission="autoplays:write"
          verb="switch auto plays on or off"
          label={`${autoPlayTitle(autoPlay, subject)}, on or off`}
          value={autoPlay.enabled}
          disabled={update.isPending || (unpicked && !autoPlay.enabled)}
          onValueChange={(enabled) => update.mutate({ autoPlay, enabled })}
        />
      </View>
      {update.error && <ErrorDisplay error={update.error} className="px-0" />}
    </View>
  );
};
