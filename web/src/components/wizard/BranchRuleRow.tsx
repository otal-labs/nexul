import { TriangleAlert } from "lucide-react";
import { useWatch, type Control } from "react-hook-form";

import { Button } from "@/components/ui/button";
import { AdvancedFields } from "@/components/dns/AdvancedFields";
import { FormInput } from "@/components/FormInput";
import { FormSelect } from "@/components/ticket/FormSelect";
import { FormTextarea } from "@/components/ticket/FormTextarea";
import { sharesProduction, type BranchRowsFormData } from "@/models/BranchRow";
import { exampleBranch, previewHostname } from "@/utils/BranchHostnameUtility";

interface BranchRuleRowProps {
  control: Control<BranchRowsFormData>;
  index: number;
  networkOptions: { value: string; label: string }[];
  productionNetwork: string;
  onRemove: () => void;
}

export const BranchRuleRow = ({ control, index, networkOptions, productionNetwork, onRemove }: BranchRuleRowProps) => {
  const row = useWatch({ control, name: `rows.${index}` });
  const pattern = row.pattern.trim();
  const branch = exampleBranch(pattern);
  const hostname = row.hostname.trim();
  const url = hostname ? previewHostname(hostname, pattern, branch) : "no hostname";

  return (
    <li className="relative animate-in fade-in-0 slide-in-from-bottom-1 space-y-4 py-4 duration-200 ease-out">
      <FormInput control={control} name={`rows.${index}.pattern`} label="Branch" placeholder="feature/*" />
      <FormInput control={control} name={`rows.${index}.hostname`} label="Hostname" placeholder="*.example.com" />
      <FormSelect
        control={control}
        name={`rows.${index}.network`}
        label="Network"
        placeholder="Choose a network…"
        options={networkOptions}
      />
      {pattern && (
        <p className="font-mono text-xs wrap-break-word text-muted-foreground">
          {branch} → {url} on {row.network || "no network"}
        </p>
      )}
      {sharesProduction(row, productionNetwork) && (
        <p className="flex gap-2 text-sm">
          <TriangleAlert aria-hidden className="mt-0.5 size-4 shrink-0 text-warning" />
          Uses production&apos;s services, including its database — testers are never sent here.
        </p>
      )}
      <AdvancedFields>
        <FormInput control={control} name={`rows.${index}.port`} label="Port" inputMode="numeric" placeholder="8080" />
        <FormTextarea
          control={control}
          name={`rows.${index}.overrides`}
          label="Overrides"
          placeholder="DATABASE_URL=postgres://qa-db/app"
          rows={3}
          className="bg-background font-mono text-xs"
        />
        <p className="text-xs text-muted-foreground">
          One KEY=value per line, replacing the default branch&apos;s values for this branch only.
        </p>
      </AdvancedFields>
      <Button
        type="button"
        variant="ghost"
        size="sm"
        className="absolute top-3 right-0 -mr-2 text-muted-foreground"
        aria-label={`Remove ${pattern || "this branch"}`}
        onClick={onRemove}
      >
        Remove
      </Button>
    </li>
  );
};
