import type { PermissionAction } from "@nexul/client-core/permissionActions";
import type { EntityAnswer } from "@nexul/client-core/actionState";

import { useEntityActions } from "@/hooks/useEntityActions";
import type { Stack } from "@/models/Stack";

// An instance stack answers to its instance area and a branch deployment to its base stack, so neither is explained.
export const useStackActions = (stack: Pick<Stack, "id" | "project_id" | "derived_from">): ((permission: PermissionAction) => EntityAnswer | undefined) =>
  useEntityActions("stack", stack.id, !!stack.project_id && !stack.derived_from);
