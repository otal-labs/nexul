import { zodResolver } from "@hookform/resolvers/zod";
import { Plus } from "lucide-react";
import { useFieldArray, useForm } from "react-hook-form";

import { Button } from "@/components/ui/button";
import { WizardFooter } from "@/components/wizard/WizardFooter";
import { WizardSkipLink } from "@/components/wizard/WizardSkipLink";
import { BranchRuleRow } from "@/components/wizard/BranchRuleRow";
import { DefaultBranchRow } from "@/components/wizard/DefaultBranchRow";
import { useUpdateStack } from "@/hooks/StackHooks";
import { useProjectWizardStore } from "@/stores/projectWizardStore";
import { branchRowsFormSchema, ruleFromRow, rowFromRule, type BranchRowsFormData } from "@/models/BranchRow";
import type { Exposure } from "@/models/DNS";
import { defaultNetwork, type BranchDeployRule, type Stack } from "@/models/Stack";
import type { MachineNetwork } from "@/utils/MachineNetworkUtility";

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

  const gatewayNetworks = new Set(networks.filter((n) => n.hasGateway).map((n) => n.name));

  const form = useForm<BranchRowsFormData>({
    defaultValues: {
      // On for a fresh stack; a saved rule set without the default branch's rule means the owner switched it off.
      deployDefault: rules.length === 0 || rules.some(isDefaultRule),
      rows: rules.filter((r) => !isDefaultRule(r)).map((r) => rowFromRule(r, defaultPort)),
    },
    resolver: zodResolver(branchRowsFormSchema(gatewayNetworks)),
  });
  const { fields, append, remove } = useFieldArray({ control: form.control, name: "rows" });

  const addRow = () =>
    append({
      pattern: "",
      hostname: exposure ? `*.${exposure.zone}` : "",
      network: productionNetwork,
      port: defaultPort,
      overrides: "",
    });

  const onSubmit = async (data: BranchRowsFormData) => {
    const next = [
      ...(data.deployDefault ? [defaultRule] : []),
      ...data.rows.map((row) => ruleFromRow(row, gatewayNetworks.has(row.network))),
    ];
    try {
      await updateStack.mutateAsync({ ...stack, branch_deploy_rules: next });
      setBranchesSummary(next.length > 0 ? next.map((r) => r.pattern).join(", ") : "No branches deploy on push");
      onDone();
    } catch {
      // Errors surface through the hook's toast; saving the rules is retry-safe.
    }
  };

  return (
    <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-5">
      <ul className="divide-y divide-border border-y border-border">
        <DefaultBranchRow
          control={form.control}
          branch={defaultBranch}
          hostname={exposure?.hostname}
          network={productionNetwork}
        />
        {fields.map((field, index) => (
          <BranchRuleRow
            key={field.id}
            control={form.control}
            index={index}
            networks={networks}
            productionNetwork={productionNetwork}
            onRemove={() => remove(index)}
          />
        ))}
      </ul>
      <Button type="button" variant="outline" size="sm" onClick={addRow}>
        <Plus aria-hidden className="size-4" />
        Add branch
      </Button>
      <WizardFooter skip={<WizardSkipLink onClick={onSkip} />}>
        <Button type="submit" loading={updateStack.isPending}>
          Save branches
        </Button>
      </WizardFooter>
    </form>
  );
};
