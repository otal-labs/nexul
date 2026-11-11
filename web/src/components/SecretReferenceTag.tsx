import { KeyRoundIcon } from "lucide-react";

import { parseSecretReference, secretManagerName } from "@nexul/client-core/secretReference";

import { NoFillBadge } from "@/components/ui/badge";

interface SecretReferenceTagProps {
  value: string;
}

// Marks a value that names a secret in an external manager instead of holding it; renders nothing for a literal or a malformed one.
export const SecretReferenceTag = ({ value }: SecretReferenceTagProps) => {
  const parsed = parseSecretReference(value);
  if (parsed.kind !== "reference") return null;
  const manager = secretManagerName(parsed.ref.scheme);
  return (
    <NoFillBadge icon={KeyRoundIcon} color="text-muted-foreground" className="shrink-0" title={`Read from ${manager} when used`}>
      {manager}
    </NoFillBadge>
  );
};
