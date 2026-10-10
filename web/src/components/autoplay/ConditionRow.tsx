import { useFormContext } from "react-hook-form";

import { RemoveIcon } from "@/components/autoplay/ComposerControls";
import { ConditionFields, type RulePath } from "@/components/autoplay/ConditionFields";
import type { AutoPlayDraft } from "@/models/AutoPlay";

interface ConditionRowProps {
  path: RulePath;
  onRemove: () => void;
  // The muted word joining a rule to the one above it ("and", "or").
  lead?: string;
}

export const ConditionRow = ({ path, onRemove, lead }: ConditionRowProps) => {
  const { formState } = useFormContext<AutoPlayDraft>();
  return (
    <div className="flex items-center gap-1.5">
      {lead && <span className="text-sm text-muted-foreground">{lead}</span>}
      <ConditionFields path={path} />
      {!formState.disabled && <RemoveIcon label="Remove condition" onClick={onRemove} />}
    </div>
  );
};
