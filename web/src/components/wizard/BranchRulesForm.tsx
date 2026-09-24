import { zodResolver } from "@hookform/resolvers/zod";
import { Plus } from "lucide-react";
import { useFieldArray, useForm } from "react-hook-form";

import { Button } from "@/components/ui/button";
import { BranchRuleRow } from "@/components/wizard/BranchRuleRow";
import { DefaultBranchRow } from "@/components/wizard/DefaultBranchRow";
import { useUpdateStack } from "@/hooks/StackHooks";
import { useProjectWizardStore } from "@/stores/projectWizardStore";
import { BranchRowsFormSchema, ruleFromRow, rowFromRule, type BranchRowsFormData } from "@/models/BranchRow";
import type { Exposure } from "@/models/DNS";
import { defaultNetwork, type BranchDeployRule, type Stack } from "@/models/Stack";
import { networkLabel, type MachineNetwork } from "@/utils/MachineNetworkUtility";

interface BranchRulesFormProps {
  stack: Stack;
  exposure: Exposure | undefined;
  defaultPort: string;
  networks: MachineNetwork[];
  onDone: () => void;
  onSkip: () => void;
}

// Saving replaces the stack's rules with the rows, so reopening the rung edits the same set.
export const BranchRulesForm = ({ stack, exposure, defaultPort, networks, onDone, onSkip }: BranchRulesFormProps) => {
  const setBranchesSummary = useProjectWizardStore((s) => s.setBranchesSummary);
  const updateStack = useUpdateStack();
  const productionNetwork = defaultNetwork(stack);
  const defaultBranch = stack.build_source?.branch || "main";
  const rules = stack.branch_deploy_rules ?? [];
  const isDefaultRule = (r: BranchDeployRule) => r.pattern === defaultBranch && !r.name_suffix;
  const defaultRule = rules.find(isDefaultRule) ?? { pattern: defaultBranch, docker_network: productionNetwork };

  const form = useForm<BranchRowsFormData>({
    defaultValues: { rows: rules.filter((r) => !isDefaultRule(r)).map((r) => rowFromRule(r, defaultPort)) },
    resolver: zodResolver(BranchRowsFormSchema),
  });
  const { fields, append, remove } = useFieldArray({ control: form.control, name: "rows" });
  const networkOptions = networks.map((n) => ({ value: n.name, label: networkLabel(n) }));

  const addRow = () =>
    append({
      pattern: "",
      hostname: exposure ? `*.${exposure.zone}` : "",
      network: productionNetwork,
      port: defaultPort,
      overrides: "",
    });

  const onSubmit = async (data: BranchRowsFormData) => {
    const next = [defaultRule, ...data.rows.map(ruleFromRow)];
    try {
      await updateStack.mutateAsync({ ...stack, branch_deploy_rules: next });
      setBranchesSummary(next.map((r) => r.pattern).join(", "));
      onDone();
    } catch {
      // Errors surface through the hook's toast; saving the rules is retry-safe.
    }
  };

  return (
    <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-5">
      <ul className="divide-y divide-border border-y border-border">
        <DefaultBranchRow branch={defaultBranch} hostname={exposure?.hostname} network={productionNetwork} />
        {fields.map((field, index) => (
          <BranchRuleRow
            key={field.id}
            control={form.control}
            index={index}
            networkOptions={networkOptions}
            productionNetwork={productionNetwork}
            onRemove={() => remove(index)}
          />
        ))}
      </ul>
      <Button type="button" variant="outline" size="sm" onClick={addRow}>
        <Plus aria-hidden className="size-4" />
        Add branch
      </Button>
      <div className="flex flex-wrap gap-3">
        <Button type="submit" className="w-full sm:w-auto" disabled={updateStack.isPending}>
          {updateStack.isPending ? "Saving…" : "Save branches"}
        </Button>
        <Button type="button" variant="ghost" className="text-muted-foreground" onClick={onSkip}>
          Skip for now
        </Button>
      </div>
    </form>
  );
};
