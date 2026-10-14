import { answerFor, type EntityAnswer } from "@nexul/client-core/actionState";
import type { PermissionAction } from "@nexul/client-core/permissionActions";

import { useFetchExplanation } from "@/hooks/PermissionHooks";

export type EntityKind = "doc" | "play" | "project" | "stack";

// The viewer's answer per action from the one explain request a page's components share; no answer keeps the role-wide one.
export const useEntityActions = (kind: EntityKind, id: string, enabled = true): ((permission: PermissionAction) => EntityAnswer | undefined) => {
  const { data } = useFetchExplanation(kind, id, "", enabled);
  return (permission) => answerFor(data, permission);
};
