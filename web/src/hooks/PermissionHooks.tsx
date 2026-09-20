import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";

import { api, errorMessage } from "@/api/client";
import { getDocsKey } from "@/hooks/DocHooks";
import type { PermissionGrant, PermissionInfo, PermissionUser, SetPermissionsInput } from "@/models/Permission";

const getGrantsKey = (resourceType: string, resourceId: string) => ["getGrants", resourceType, resourceId];
const getPermissionUsersKey = "getPermissionUsers";
const getPermissionCatalogKey = "getPermissionCatalog";

// Backs a play's excluded-users count (ticket 21). enabled defaults on; a caller without permissions:write
// (which the endpoint requires) passes false so the row never fires a request it knows will 403.
export const useFetchGrants = (resourceType: string, resourceId: string, enabled = true) =>
  useQuery({
    queryKey: getGrantsKey(resourceType, resourceId),
    queryFn: async () =>
      (
        await api.get<{ grants: PermissionGrant[] }>(
          `/api/permissions?resource_type=${resourceType}&resource_id=${resourceId}`,
        )
      ).data.grants,
    enabled: enabled && resourceId !== "",
  });

export const useFetchPermissionUsers = () =>
  useQuery({
    queryKey: [getPermissionUsersKey],
    queryFn: async () => (await api.get<{ users: PermissionUser[] }>("/api/permissions/users")).data.users,
  });

// The one source of the permission vocabulary — every grid/checklist in the app renders this, never a hardcoded list.
export const useFetchPermissionCatalog = () =>
  useQuery({
    queryKey: [getPermissionCatalogKey],
    queryFn: async () =>
      (await api.get<{ permissions: PermissionInfo[] }>("/api/permissions/catalog")).data.permissions,
    staleTime: Infinity,
  });

type PlayPermissionsInput = Extract<SetPermissionsInput, { resource_type: "play" }>;

const isPlayInput = (input: SetPermissionsInput): input is PlayPermissionsInput =>
  "resource_type" in input && input.resource_type === "play";

export const useSetPermissions = () => {
  const client = useQueryClient();
  return useMutation({
    mutationFn: async (input: SetPermissionsInput) => {
      await api.put("/api/permissions", input);
      return input;
    },
    onSuccess: async (input) => {
      if (isPlayInput(input)) {
        for (const id of input.resource_ids) {
          await client.invalidateQueries({ queryKey: getGrantsKey("play", id) });
        }
        toast.success(input.grant ? "User un-excluded" : "User excluded");
        return;
      }
      for (const id of input.doc_ids) {
        await client.invalidateQueries({ queryKey: getGrantsKey("doc", id) });
      }
      await client.invalidateQueries({ queryKey: [getDocsKey] });
      toast.success(input.grant ? "Permissions granted" : "Permissions revoked");
    },
    onError: (error) => toast.error(errorMessage(error)),
  });
};
