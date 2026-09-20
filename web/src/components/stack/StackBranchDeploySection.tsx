import { useState } from "react";
import { zodResolver } from "@hookform/resolvers/zod";
import { useForm } from "react-hook-form";

import { EmptyRow } from "@/components/EmptyRow";
import { FormInput } from "@/components/FormInput";
import { SettingsCard } from "@/components/settings/SettingsCard";
import { Button } from "@/components/ui/button";
import { useDeleteStack, useUpdateStack } from "@/hooks/StackHooks";
import { RuleFormSchema, type BranchDeployRule, type RuleFormData, type StackWithBranches } from "@/models/Stack";

interface StackBranchDeploySectionProps {
  stack: StackWithBranches;
}

const microheader = "font-mono text-[11px] font-medium tracking-[0.18em] text-muted-foreground uppercase";

// A derived stack (stack.branch set) never gets this section — it has no rules of its own.
export const StackBranchDeploySection = ({ stack }: StackBranchDeploySectionProps) => {
  const updateStack = useUpdateStack();
  const deleteStack = useDeleteStack();
  const [adding, setAdding] = useState(false);
  const form = useForm<RuleFormData>({
    defaultValues: { pattern: "", docker_network: "", hostname_template: "", name_suffix: "", port: "" },
    resolver: zodResolver(RuleFormSchema),
  });

  const rules = stack.branch_deploy_rules ?? [];
  const branchDeployments = stack.branch_deployments ?? [];

  const saveRules = (next: BranchDeployRule[]) => updateStack.mutateAsync({ ...stack, branch_deploy_rules: next });

  const addRule = async (data: RuleFormData) => {
    const port = data.port.trim() ? Number(data.port.trim()) : undefined;
    const rule: BranchDeployRule = {
      pattern: data.pattern,
      docker_network: data.docker_network,
      ...(data.hostname_template ? { hostname_template: data.hostname_template } : {}),
      ...(data.name_suffix ? { name_suffix: data.name_suffix } : {}),
      ...(port ? { port } : {}),
    };
    await saveRules([...rules, rule]);
    form.reset();
    setAdding(false);
  };

  const removeRule = (index: number) => saveRules(rules.filter((_, i) => i !== index));

  const cancel = () => {
    form.reset();
    setAdding(false);
  };

  return (
    <SettingsCard
      id="branch-deploys"
      title="Branch deploys"
      description="A push to a matching branch deploys it as its own copy of this stack, on its own network and hostname."
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
            <Button size="sm" variant="ghost" onClick={cancel}>
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
                <li key={`${rule.pattern}-${i}`} className="flex items-center justify-between gap-3 px-3 py-2.5 text-xs">
                  <div className="min-w-0 space-y-0.5">
                    <p className="font-mono">{rule.pattern}</p>
                    <p className="truncate text-muted-foreground">
                      {rule.docker_network}
                      {rule.hostname_template && ` → ${rule.hostname_template}${rule.port ? `:${rule.port}` : ""}`}
                    </p>
                  </div>
                  <Button variant="ghost" size="sm" onClick={() => removeRule(i)} disabled={updateStack.isPending}>
                    Remove
                  </Button>
                </li>
              ))}
            </ul>
          )}
          {adding && (
            <form
              onSubmit={form.handleSubmit(addRule)}
              className="animate-in fade-in-0 slide-in-from-bottom-1 space-y-3 rounded-lg border border-border p-4 duration-150 ease-out"
            >
              <FormInput control={form.control} name="pattern" label="Branch pattern" placeholder="feature/*" />
              <FormInput control={form.control} name="docker_network" label="Docker network" placeholder="app-net" />
              <FormInput
                control={form.control}
                name="hostname_template"
                label="Hostname template (optional)"
                placeholder="{branch}.example.com"
              />
              <FormInput
                control={form.control}
                name="port"
                label="Port (required with a hostname template)"
                placeholder="8080"
              />
              <FormInput
                control={form.control}
                name="name_suffix"
                label="Name suffix (exact patterns only, optional)"
                placeholder="qa"
              />
              <Button type="submit" size="sm" disabled={updateStack.isPending}>
                {updateStack.isPending ? "Saving…" : "Save rule"}
              </Button>
            </form>
          )}
        </div>

        <div className="space-y-2">
          <p className={microheader}>Live branch deployments</p>
          {branchDeployments.length === 0 && <EmptyRow>No branch deployments yet.</EmptyRow>}
          {branchDeployments.length > 0 && (
            <ul className="divide-y divide-border rounded-lg border border-border">
              {branchDeployments.map((d) => (
                <li key={d.id} className="flex items-center justify-between gap-3 px-3 py-2.5 text-xs">
                  <div className="min-w-0 space-y-0.5">
                    <p className="font-mono">{d.name}</p>
                    <p className="text-muted-foreground">branch {d.branch}</p>
                  </div>
                  <Button variant="outline" size="sm" onClick={() => deleteStack.mutate(d.id)} disabled={deleteStack.isPending}>
                    Tear down
                  </Button>
                </li>
              ))}
            </ul>
          )}
        </div>
      </div>
    </SettingsCard>
  );
};
