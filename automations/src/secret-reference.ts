// Recognises a secret reference the way internal/platform/secretref does (ADR 0181); a test runs that package's table.
export type ReferenceKind = "literal" | "reference" | "malformed";

export interface SecretReference {
  scheme: string;
  vault: string;
  item: string;
  section?: string;
  field: string;
}

// The closed list of managers; a later manager adds its prefix here.
const SCHEMES = ["op://"];

// Go's strings.TrimSpace trims unicode.IsSpace, which is the White_Space property.
const PADDED = /^\p{White_Space}|\p{White_Space}$/u;

export function parseSecretReference(value: string): { kind: ReferenceKind; ref?: SecretReference } {
  for (const prefix of SCHEMES) {
    if (!value.startsWith(prefix)) continue;
    return parseOp(prefix.slice(0, -3), value.slice(prefix.length));
  }
  return { kind: "literal" };
}

function parseOp(scheme: string, rest: string): { kind: ReferenceKind; ref?: SecretReference } {
  const parts = rest.split("/");
  if (parts.length < 3 || parts.length > 4) return { kind: "malformed" };
  if (parts.some((p) => p === "" || PADDED.test(p))) return { kind: "malformed" };
  const ref: SecretReference = { scheme, vault: parts[0] ?? "", item: parts[1] ?? "", field: parts[parts.length - 1] ?? "" };
  if (parts.length === 4) ref.section = parts[2] ?? "";
  return { kind: "reference", ref };
}
