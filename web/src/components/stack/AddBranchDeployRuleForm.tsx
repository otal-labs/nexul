import { zodResolver } from "@hookform/resolvers/zod";
import { useForm, useWatch } from "react-hook-form";

import { Button } from "@/components/ui/button";
import { FormInput } from "@/components/FormInput";
import { FormTextarea } from "@/components/ticket/FormTextarea";
import { useUpdateStack } from "@/hooks/StackHooks";
import { derivesClone, RuleFormSchema, type BranchDeployRule, type RuleFormData, type Stack } from "@/models/Stack";
import { parseEnv } from "@/lib/env";

interface AddBranchDeployRuleFormProps {
  stack: Stack;
  onDone: () => void;
}

export const AddBranchDeployRuleForm = ({ stack, onDone }: AddBranchDeployRuleFormProps) => {
  const updateStack = useUpdateStack();
  const form = useForm<RuleFormData>({
    defaultValues: { pattern: "", docker_network: "", hostname_template: "", name_suffix: "", port: "", overrides: "" },
    resolver: zodResolver(RuleFormSchema),
  });
  const [pattern, nameSuffix] = useWatch({ control: form.control, name: ["pattern", "name_suffix"] });
  const canOverride = derivesClone({ pattern, name_suffix: nameSuffix });

  const addRule = async (data: RuleFormData) => {
    const port = data.port ? Number(data.port) : undefined;
    const overrides = parseEnv(data.overrides);
    const rule: BranchDeployRule = {
      pattern: data.pattern,
      docker_network: data.docker_network,
      ...(data.hostname_template ? { hostname_template: data.hostname_template } : {}),
      ...(data.name_suffix ? { name_suffix: data.name_suffix } : {}),
      ...(port ? { port } : {}),
      ...(canOverride && Object.keys(overrides).length > 0 ? { overrides } : {}),
    };
    await updateStack.mutateAsync({ ...stack, branch_deploy_rules: [...(stack.branch_deploy_rules ?? []), rule] });
    onDone();
  };

  return (
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
      <FormInput control={form.control} name="port" label="Port (required with a hostname template)" placeholder="8080" />
      <FormInput
        control={form.control}
        name="name_suffix"
        label="Name suffix (exact patterns only, optional)"
        placeholder="qa"
      />
      {canOverride && (
        <FormTextarea
          control={form.control}
          name="overrides"
          label="Overrides (optional)"
          placeholder="DATABASE_URL=postgres://qa-db/app"
          rows={3}
          className="bg-background font-mono text-xs"
        />
      )}
      <Button type="submit" size="sm" disabled={updateStack.isPending}>
        {updateStack.isPending ? "Saving…" : "Save rule"}
      </Button>
    </form>
  );
};
