import { useState } from "react";

import { Button } from "@/components/ui/button";
import { EmptyRow } from "@/components/EmptyRow";
import { Microheader } from "@/components/Microheader";
import { SettingsCard } from "@/components/settings/SettingsCard";
import { AddBranchDeployRuleForm } from "@/components/stack/AddBranchDeployRuleForm";
import { BranchDeployRuleRow } from "@/components/stack/BranchDeployRuleRow";
import { BranchDeploymentRow } from "@/components/stack/BranchDeploymentRow";
import type { StackWithBranches } from "@/models/Stack";

interface StackBranchDeploySectionProps {
  stack: StackWithBranches;
}

// A derived stack (stack.branch set) never gets this section — it has no rules of its own.
export const StackBranchDeploySection = ({ stack }: StackBranchDeploySectionProps) => {
  const [adding, setAdding] = useState(false);

  const rules = stack.branch_deploy_rules ?? [];
  const branchDeployments = stack.branch_deployments ?? [];

  return (
    <SettingsCard
      id="branch-deploys"
      title="Branch deploys"
      description="A push to a matching branch deploys it as its own copy of this stack, on its own network and hostname, with any settings the rule overrides."
      footer={
        <>
          <p className="text-xs text-muted-foreground">
            {rules.length === 1 && "1 rule."}
            {rules.length > 1 && `${rules.length} rules.`}
          </p>
          {!adding && (
            <Button size="sm" onClick={() => setAdding(true)}>
              Add rule
            </Button>
          )}
          {adding && (
            <Button size="sm" variant="ghost" onClick={() => setAdding(false)}>
              Cancel
            </Button>
          )}
        </>
      }
    >
      <div className="space-y-6">
        <div className="space-y-2">
          <Microheader>Rules</Microheader>
          {rules.length === 0 && !adding && (
            <EmptyRow className="px-0 py-0">No rules yet, so every branch is ignored until one matches.</EmptyRow>
          )}
          {rules.length > 0 && (
            <ul className="divide-y divide-border rounded-lg border border-border">
              {rules.map((rule, i) => (
                <BranchDeployRuleRow key={`${rule.pattern}-${i}`} stack={stack} rule={rule} index={i} />
              ))}
            </ul>
          )}
          {adding && <AddBranchDeployRuleForm stack={stack} onDone={() => setAdding(false)} />}
        </div>

        <div className="space-y-2">
          <Microheader>Live branch deployments</Microheader>
          {branchDeployments.length === 0 && <EmptyRow className="px-0 py-0">No branch deployments yet.</EmptyRow>}
          {branchDeployments.length > 0 && (
            <ul className="divide-y divide-border rounded-lg border border-border">
              {branchDeployments.map((d) => (
                <BranchDeploymentRow key={d.id} deployment={d} />
              ))}
            </ul>
          )}
        </div>
      </div>
    </SettingsCard>
  );
};
