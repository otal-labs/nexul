import { zodResolver } from "@hookform/resolvers/zod";
import { useForm } from "react-hook-form";

import { Button } from "@/components/ui/button";
import { FormTextarea } from "@/components/ticket/FormTextarea";
import { RuleOverridesFormSchema, type RuleOverridesFormData } from "@/models/Stack";
import { formatEnv, parseEnv } from "@/lib/env";

interface BranchDeployRuleOverridesFormProps {
  overrides: Record<string, string> | undefined;
  saving: boolean;
  onSave: (overrides: Record<string, string>) => Promise<unknown>;
  onCancel: () => void;
}

export const BranchDeployRuleOverridesForm = ({ overrides, saving, onSave, onCancel }: BranchDeployRuleOverridesFormProps) => {
  const form = useForm<RuleOverridesFormData>({
    defaultValues: { overrides: formatEnv(overrides) },
    resolver: zodResolver(RuleOverridesFormSchema),
  });

  return (
    <form
      onSubmit={form.handleSubmit((data) => onSave(parseEnv(data.overrides)))}
      className="animate-in fade-in-0 slide-in-from-top-1 space-y-3 duration-150 ease-out"
    >
      <FormTextarea
        control={form.control}
        name="overrides"
        label="Overrides"
        placeholder="DATABASE_URL=postgres://qa-db/app"
        rows={3}
        className="bg-background font-mono text-xs"
      />
      <p className="text-xs text-muted-foreground">
        One KEY=value per line, replacing the base stack&apos;s value for this branch only. Remove a line and the next
        deploy uses the base value again.
      </p>
      <div className="flex gap-2">
        <Button type="submit" size="sm" disabled={saving}>
          {saving ? "Saving…" : "Save overrides"}
        </Button>
        <Button type="button" size="sm" variant="ghost" onClick={onCancel}>
          Cancel
        </Button>
      </div>
    </form>
  );
};
