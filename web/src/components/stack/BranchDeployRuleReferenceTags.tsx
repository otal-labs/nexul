import { useWatch, type Control } from "react-hook-form";

import { parseSecretReference } from "@nexul/client-core/secretReference";

import { SecretReferenceTag } from "@/components/SecretReferenceTag";
import { parseEnv } from "@/lib/env";
import type { RuleOverridesFormData } from "@/models/Stack";

interface BranchDeployRuleReferenceTagsProps {
  control: Control<RuleOverridesFormData>;
}

// One line per override whose value is a secret reference, since the textarea has no room beside a single line.
export const BranchDeployRuleReferenceTags = ({ control }: BranchDeployRuleReferenceTagsProps) => {
  const text = useWatch({ control, name: "overrides" });
  const references = Object.entries(parseEnv(text)).filter(([, value]) => parseSecretReference(value).kind === "reference");

  return (
    references.length > 0 && (
      <ul className="space-y-1">
        {references.map(([key, value]) => (
          <li key={key} className="flex items-center gap-2 font-mono text-xs">
            {key}
            <SecretReferenceTag value={value} />
          </li>
        ))}
      </ul>
    )
  );
};
