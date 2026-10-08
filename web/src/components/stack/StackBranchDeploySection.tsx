import { useState } from "react";

import { Button } from "@/components/ui/button";
import { EnterList } from "@/components/EnterList";
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
      description="A push to a matching branch deploys its own copy of this stack, with its own network, hostname, and overrides."
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
            <EmptyRow flush>No rules yet, so no branch deploys. Add one to deploy matching branches.</EmptyRow>
          )}
          {rules.length > 0 && (
            <EnterList className="divide-y divide-border rounded-lg border border-border">
              {rules.map((rule, i) => (
                <BranchDeployRuleRow key={`${rule.pattern}-${i}`} stack={stack} rule={rule} index={i} />
              ))}
            </EnterList>
          )}
          {adding && <AddBranchDeployRuleForm stack={stack} onDone={() => setAdding(false)} />}
        </div>

        <div className="space-y-2">
          <Microheader>Live branch deployments</Microheader>
          {branchDeployments.length === 0 && <EmptyRow flush>No branch deployments yet.</EmptyRow>}
          {branchDeployments.length > 0 && (
            <EnterList className="divide-y divide-border rounded-lg border border-border">
              {branchDeployments.map((d) => (
                <BranchDeploymentRow key={d.id} deployment={d} />
              ))}
            </EnterList>
          )}
        </div>
      </div>
    </SettingsCard>
  );
};
