import { RuleStateControl, lockedAllowReason, lockedLiftReason } from "@/components/access/RuleStateControl";
import { SourceLine } from "@/components/access/SourceLine";
import { usePermissionRules } from "@/hooks/usePermissionRules";
import { RuleState, type PermissionAnswer } from "@/models/Permission";

interface PermissionRuleRowProps {
  userId: string;
  action: string;
  label: string;
  // The person's answer here; its fallback is what applies without this rule.
  answer: PermissionAnswer | undefined;
}

// One permission: its name, the three-state control, and under Fallback what applies without a rule here.
export const PermissionRuleRow = ({ userId, action, label, answer }: PermissionRuleRowProps) => {
  const { stateOf, saved, holds, setState } = usePermissionRules();
  const value = stateOf(userId, action);
  const held = holds(action);
  const liftLocked = !held && saved(userId, action) === RuleState.Deny;
  const locked = {
    ...(!held ? { [RuleState.Allow]: lockedAllowReason } : {}),
    ...(liftLocked ? { [RuleState.Fallback]: lockedLiftReason } : {}),
  };

  return (
    <li className="flex items-start justify-between gap-4 px-3 py-2.5">
      <div className="min-w-0 flex-1 pt-1">
        <p className="text-sm">{label}</p>
        {value === RuleState.Fallback && answer && <SourceLine verdict={answer.fallback} className="mt-1" />}
      </div>
      <RuleStateControl label={label} value={value} locked={locked} onChange={(state) => setState(userId, action, state)} />
    </li>
  );
};
