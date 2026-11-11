// Mirrors internal/platform/secretref: a value that names, in full, where a secret lives in an external secrets manager.
export interface SecretRef {
  scheme: string;
  vault: string;
  item: string;
  section?: string;
  field: string;
}

export type ParsedSecret = { kind: "literal" } | { kind: "reference"; ref: SecretRef } | { kind: "malformed" };

const literal: ParsedSecret = { kind: "literal" };
const malformed: ParsedSecret = { kind: "malformed" };

// The closed list of managers; a later manager adds its prefix and its display name.
const managers: Record<string, string> = { op: "1Password" };

// Go's strings.TrimSpace trims the White_Space property, which String.prototype.trim does not match exactly.
const padded = /^\p{White_Space}|\p{White_Space}$/u;

export const parseSecretReference = (value: string): ParsedSecret => {
  for (const scheme of Object.keys(managers)) {
    const prefix = `${scheme}://`;
    if (!value.startsWith(prefix)) continue;
    return parseOp(scheme, value.slice(prefix.length));
  }
  return literal;
};

const parseOp = (scheme: string, rest: string): ParsedSecret => {
  const parts = rest.split("/");
  if (parts.length < 3 || parts.length > 4) return malformed;
  if (parts.some((p) => p === "" || padded.test(p))) return malformed;
  const [vault, item] = parts as [string, string];
  const field = parts[parts.length - 1] as string;
  const ref: SecretRef = { scheme, vault, item, field };
  if (parts.length === 4) ref.section = parts[2] as string;
  return { kind: "reference", ref };
};

export const secretManagerName = (scheme: string): string => managers[scheme] ?? scheme;

// The server's refusal opens with the name of the field that held the reference; this finds which of names it was.
export const holderNamedBy = (message: string, names: string[]): string | undefined =>
  names.find((name) => message.startsWith(`${name} is not a valid secret reference`));

// A save the server refused because a value is a malformed reference; the editor that holds the value shows it under the field.
export const isReferenceRefusal = (message: string): boolean => / is not a valid secret reference/.test(message);
