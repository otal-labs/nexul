import { Controller, useWatch, type Control } from "react-hook-form";

import { Switch } from "@/components/ui/switch";
import type { BranchRowsFormData } from "@/models/BranchRow";

interface DefaultBranchRowProps {
  control: Control<BranchRowsFormData>;
  branch: string;
  hostname: string | undefined;
  network: string;
}

// Switching this on is the owner's consent to redeploy production on every push (ADR 0036); off saves no rule.
export const DefaultBranchRow = ({ control, branch, hostname, network }: DefaultBranchRowProps) => {
  const deployDefault = useWatch({ control, name: "deployDefault" });

  return (
    <li className="space-y-2 py-4">
      <div className="flex items-baseline justify-between gap-3">
        <p className="font-mono text-sm">{branch}</p>
        <p className="text-xs text-muted-foreground">Production</p>
      </div>
      <label className="flex items-center gap-2 text-sm">
        <Controller
          control={control}
          name="deployDefault"
          render={({ field }) => <Switch checked={field.value} onCheckedChange={field.onChange} onBlur={field.onBlur} />}
        />
        Every push to {branch} redeploys it
      </label>
      {deployDefault && (
        <p className="font-mono text-xs wrap-break-word text-muted-foreground">
          {branch} → {hostname ?? "no hostname"} on {network}
        </p>
      )}
    </li>
  );
};
