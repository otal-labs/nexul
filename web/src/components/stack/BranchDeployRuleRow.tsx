import { useState } from "react";

import { Button } from "@/components/ui/button";
import { BranchDeployRuleOverridesForm } from "@/components/stack/BranchDeployRuleOverridesForm";
import { useUpdateStack } from "@/hooks/StackHooks";
import { derivesClone, type BranchDeployRule, type Stack } from "@/models/Stack";

interface BranchDeployRuleRowProps {
  stack: Stack;
  rule: BranchDeployRule;
  index: number;
}

export const BranchDeployRuleRow = ({ stack, rule, index }: BranchDeployRuleRowProps) => {
  const updateStack = useUpdateStack();
  const [editing, setEditing] = useState(false);
  const rules = stack.branch_deploy_rules ?? [];
  const overrideKeys = Object.keys(rule.overrides ?? {}).sort();

  const saveOverrides = async (overrides: Record<string, string>) => {
    const next: BranchDeployRule = { ...rule };
    delete next.overrides;
    if (Object.keys(overrides).length > 0) next.overrides = overrides;
    await updateStack.mutateAsync({ ...stack, branch_deploy_rules: rules.map((r, i) => (i === index ? next : r)) });
    setEditing(false);
  };

  const remove = () => updateStack.mutate({ ...stack, branch_deploy_rules: rules.filter((_, i) => i !== index) });

  return (
    <li className="@container space-y-3 px-3 py-2.5 text-xs">
      <div className="flex flex-col gap-2 @sm:flex-row @sm:items-center @sm:justify-between @sm:gap-3">
        <div className="min-w-0 space-y-0.5">
          <p className="font-mono">{rule.pattern}</p>
          <p className="truncate text-muted-foreground">
            {rule.docker_network}
            {rule.hostname_template && ` → ${rule.hostname_template}${rule.port ? `:${rule.port}` : ""}`}
          </p>
          {overrideKeys.length > 0 && (
            <p className="truncate font-mono text-muted-foreground">overrides {overrideKeys.join(", ")}</p>
          )}
        </div>
        <div className="flex shrink-0 gap-1">
          {derivesClone(rule) && !editing && (
            <Button variant="ghost" size="sm" onClick={() => setEditing(true)}>
              Overrides
            </Button>
          )}
          <Button variant="ghost" size="sm" onClick={remove} disabled={updateStack.isPending}>
            Remove
          </Button>
        </div>
      </div>
      {editing && (
        <BranchDeployRuleOverridesForm
          overrides={rule.overrides}
          saving={updateStack.isPending}
          onSave={saveOverrides}
          onCancel={() => setEditing(false)}
        />
      )}
    </li>
  );
};
