import type { AutoPlayComputerRule, AutoPlayGroup, AutoPlayRule, AutoPlayRunOn } from "@nexul/client-core/autoPlayText";

// What the reach of an auto play reads of a computer.
export interface ReachableComputer {
  id: string;
  labels?: string[] | undefined;
  derived_labels?: string[] | undefined;
  plays_off?: boolean | undefined;
  auto_plays_off?: boolean | undefined;
  unfinished?: boolean | undefined;
}

// The parts of an auto play (or its draft) that say whose computers it lands on.
export interface AutoPlayWhose {
  run_on: AutoPlayRunOn | "computer";
  run_as: string;
  // A draft that will run as the caller on save.
  run_as_me?: boolean | undefined;
  conditions: { match: "all" | "any"; groups: AutoPlayGroup[] };
}

export interface ComputerReach<C extends ReachableComputer> {
  computer: C;
  reached: boolean;
}

// Mirrors placement: an unfinished setup, a computer off for plays or for auto plays, and the rule's labels and ids pass a computer over.
export const reaches = (computer: ReachableComputer, rule: AutoPlayComputerRule | undefined): boolean => {
  if (computer.unfinished || computer.plays_off || computer.auto_plays_off) return false;
  const labels = [...(computer.labels ?? []), ...(computer.derived_labels ?? [])];
  const labelsIn = rule?.labels_in ?? [];
  const labelsOut = rule?.labels_out ?? [];
  const ids = rule?.ids ?? [];
  if (labelsIn.length > 0 && !labelsIn.some((label) => labels.includes(label))) return false;
  if (labelsOut.some((label) => labels.includes(label))) return false;
  return ids.length === 0 || ids.includes(computer.id);
};

export const computersReached = <C extends ReachableComputer>(computers: C[], rule: AutoPlayComputerRule | undefined): ComputerReach<C>[] =>
  computers.map((computer) => ({ computer, reached: reaches(computer, rule) }));

const pinOf = (rule: AutoPlayRule, field: string): string | undefined => (rule.field === field && rule.op === "is" && rule.values?.length === 1 ? rule.values[0] : undefined);

const groupPin = (group: AutoPlayGroup, field: string): string | undefined => {
  if (group.match === "any" && group.rules.length > 1) return undefined;
  return group.rules.map((rule) => pinOf(rule, field)).find((value) => value !== undefined);
};

// The one person a condition pins the field to, when every matching ticket must have it.
const conditionPin = (conditions: AutoPlayWhose["conditions"], field: string): string | undefined => {
  if (conditions.match === "any") return conditions.groups.length === 1 && conditions.groups[0] ? groupPin(conditions.groups[0], field) : undefined;
  return conditions.groups.map((group) => groupPin(group, field)).find((value) => value !== undefined);
};

// The person whose computers it runs on, or undefined while that is only known at run time.
export const autoPlayOwner = (autoPlay: AutoPlayWhose, me: string): string | undefined => {
  const runOn = autoPlay.run_on === "computer" ? "person" : autoPlay.run_on;
  if (runOn === "person") {
    if (autoPlay.run_as_me) return me;
    return autoPlay.run_as === "" ? undefined : autoPlay.run_as;
  }
  if (runOn === "developer" || runOn === "tester") return conditionPin(autoPlay.conditions, runOn);
  return undefined;
};

export type AutoPlayPlaces<C extends ReachableComputer> =
  | { kind: "computers"; computers: ComputerReach<C>[] }
  // A fixed person other than the viewer: their computers are not the viewer's to list.
  | { kind: "person"; userId: string }
  | { kind: "role"; role: "causer" | "developer" | "tester" | "nobody" };

// Where an auto play would run: the viewer's own computers that pass its rule, or the sentence saying whose they are (nobody, for a person auto play nobody is picked for).
export const autoPlayPlaces = <C extends ReachableComputer>(autoPlay: AutoPlayWhose, rule: AutoPlayComputerRule | undefined, me: string, own: C[]): AutoPlayPlaces<C> => {
  const owner = autoPlayOwner(autoPlay, me);
  if (owner === me) return { kind: "computers", computers: computersReached(own, rule) };
  if (owner !== undefined) return { kind: "person", userId: owner };
  const runOn = autoPlay.run_on === "computer" ? "person" : autoPlay.run_on;
  if (runOn === "person") return { kind: "role", role: "nobody" };
  return { kind: "role", role: runOn === "developer" || runOn === "tester" ? runOn : "causer" };
};

export type ComputerRuleMode = "any" | "labelled" | "not_labelled" | "only";

// Which of the editor's four choices a stored rule reads as; ids win, as the rule's narrowest part.
export const ruleMode = (rule: AutoPlayComputerRule): ComputerRuleMode => {
  if ((rule.ids ?? []).length > 0) return "only";
  if ((rule.labels_in ?? []).length > 0) return "labelled";
  if ((rule.labels_out ?? []).length > 0) return "not_labelled";
  return "any";
};

// The values the choice's picker shows.
export const ruleValues = (rule: AutoPlayComputerRule, mode: ComputerRuleMode): string[] => {
  if (mode === "only") return rule.ids ?? [];
  if (mode === "labelled") return rule.labels_in ?? [];
  if (mode === "not_labelled") return rule.labels_out ?? [];
  return [];
};

// Picking values changes only the choice's own part of the rule, so a rule that carries both label lists keeps the other.
export const withValues = (rule: AutoPlayComputerRule, mode: ComputerRuleMode, values: string[]): AutoPlayComputerRule => {
  if (mode === "only") return { ...rule, ids: values };
  if (mode === "labelled") return { ...rule, labels_in: values };
  if (mode === "not_labelled") return { ...rule, labels_out: values };
  return rule;
};

// Switching the choice starts a fresh rule: the old values belong to another kind of value.
export const withMode = (mode: ComputerRuleMode): AutoPlayComputerRule => {
  if (mode === "only") return { ids: [] };
  if (mode === "labelled") return { labels_in: [] };
  if (mode === "not_labelled") return { labels_out: [] };
  return {};
};
