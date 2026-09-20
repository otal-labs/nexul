import type { Candidate, DeclaredService } from "@/models/Repository";
import type { Declared } from "@/models/Stack";
import { slugify } from "@/utils/SlugUtility";

// exactOptionalPropertyTypes forbids writing an optional key with an explicit undefined value, so each optional
// field is only spread in when actually present, rather than assigned unconditionally.
const declaredEntry = (s: DeclaredService): Declared => ({
  ...(s.image && { image: s.image }),
  ...(s.build?.context && { build: s.build.context }),
  ports: s.ports.map(String),
  env_keys: s.env_keys,
});

// A compose candidate's declared map is keyed by compose service name; a Dockerfile candidate (or the wizard's
// own manual candidate, which carries no parsed service at all) gets one entry keyed by the stack's own slug.
export const declaredFrom = (candidate: Candidate, stackName: string): Record<string, Declared> => {
  if (candidate.kind === "compose") {
    return Object.fromEntries(candidate.services.map((s) => [s.name, declaredEntry(s)]));
  }
  const svc = candidate.services[0];
  return { [slugify(stackName)]: svc ? declaredEntry(svc) : {} };
};
