import { useState } from "react";

import { Button } from "@/components/ui/button";
import { EmptyRow } from "@/components/EmptyRow";
import { SettingsCard } from "@/components/settings/SettingsCard";
import { AddBranchDeployRuleForm } from "@/components/stack/AddBranchDeployRuleForm";
import { BranchDeployRuleRow } from "@/components/stack/BranchDeployRuleRow";
import { BranchDeploymentRow } from "@/components/stack/BranchDeploymentRow";
import type { StackWithBranches } from "@/models/Stack";

interface StackBranchDeploySectionProps {
  stack: StackWithBranches;
}

const microheader = "font-mono text-[11px] font-medium tracking-[0.18em] text-muted-foreground uppercase";

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
            {rules.length === 0 && "No rules yet — every branch is ignored until one matches."}
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
          <p className={microheader}>Rules</p>
          {rules.length === 0 && !adding && <EmptyRow>No branch deploy rules yet.</EmptyRow>}
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
          <p className={microheader}>Live branch deployments</p>
          {branchDeployments.length === 0 && <EmptyRow>No branch deployments yet.</EmptyRow>}
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
