import { useFormContext, useWatch } from "react-hook-form";

import { useFetchExplanation, useFetchGrants } from "@/hooks/PermissionHooks";
import { usePanelEntity } from "@/hooks/usePanelEntity";
import { usePermissionPanelStore } from "@/stores/permissionPanelStore";
import { RuleState, type PermissionRulesFormData } from "@/models/Permission";
import { ruleKey, savedState } from "@/models/PermissionPanel";

// The panel's rules as the giver sees them: what is saved, with the changes held in the dialog's form on top.
export const usePermissionRules = () => {
  const entity = usePanelEntity();
  const { data: grants } = useFetchGrants(entity.type, entity.id);
  const { data: mine } = useFetchExplanation(entity.type, entity.id);
  const { setValue } = useFormContext<PermissionRulesFormData>();
  const held = useWatch<PermissionRulesFormData, "rules">({ name: "rules" }) ?? {};
  const added = usePermissionPanelStore((s) => s.added);
  const target = usePermissionPanelStore((s) => s.target);
  const actions = mine?.effective.map((a) => a.action) ?? [];

  const saved = (userId: string, action: string) => savedState(grants?.find((g) => g.user_id === userId), action);
  const stateOf = (userId: string, action: string): RuleState => held[ruleKey(userId, action)] ?? saved(userId, action);
  const ruleCount = (userId: string) => actions.filter((a) => stateOf(userId, a) !== RuleState.Fallback).length;
  const write = (changes: Record<string, RuleState>) => setValue("rules", { ...held, ...changes }, { shouldDirty: true });

  const byAge = [...(grants ?? [])].sort((a, b) => a.created_at.localeCompare(b.created_at)).map((g) => g.user_id);
  const listed = [...new Set([...byAge, ...added])].filter((u) => ruleCount(u) > 0 || added.includes(u) || u === target);

  // Locked states for the giver: Allow on what they don't hold, and every way out of a saved Deny on it (ADR 0088).
  const holds = (action: string) => mine?.effective.find((a) => a.action === action)?.allowed ?? false;

  return {
    actions,
    listed,
    stateOf,
    saved,
    holds,
    ruleCount,
    setState: (userId: string, action: string, state: RuleState) => write({ [ruleKey(userId, action)]: state }),
    clear: (userId: string) => write(Object.fromEntries(actions.map((a) => [ruleKey(userId, a), RuleState.Fallback]))),
  };
};
