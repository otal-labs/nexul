import { parseSecretReference } from "@nexul/client-core/secretReference";

import type { Runner } from "@/models/Runner";

// The managers, by scheme, whose references the env holds.
export const referenceManagers = (env: Record<string, string> | undefined): string[] => {
  const schemes = new Set<string>();
  for (const value of Object.values(env ?? {})) {
    const parsed = parseSecretReference(value);
    if (parsed.kind === "reference") schemes.add(parsed.ref.scheme);
  }
  return [...schemes].sort();
};

// The managers of the env's references that no runner on the machine can read, as the server decides a deploy's refusal:
// a runner counts when it last reported the manager ready or unreachable. A machine whose runners the viewer cannot
// see has none in the list, so nothing is claimed about it.
export const unreadableManagers = (env: Record<string, string> | undefined, machine: string, runners: Runner[]): string[] => {
  const onMachine = runners.filter((r) => r.machine === machine);
  if (onMachine.length === 0) return [];
  return referenceManagers(env).filter(
    (scheme) => !onMachine.some((r) => r.managers?.some((m) => m.manager === scheme && m.state !== "token rejected")),
  );
};
