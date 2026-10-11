import { useMemo } from "react";
import { type QueryClient, useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";

import { api, errorMessage } from "@/api/client";
import { getDocsKey } from "@/hooks/DocHooks";
import {
  PermissionArea,
  type PermissionExplanation,
  type PermissionGrant,
  type PermissionInfo,
  type RuleChange,
} from "@/models/Permission";
import { domainsIn, type PermissionDomain } from "@/models/PermissionLevel";
import type { LiveFollower } from "@/lib/live";

export const getGrantsKey = "getGrants";
export const getExplanationKey = "getExplanation";
const getPermissionCatalogKey = "getPermissionCatalog";

// A doc's or play's rules, one row per person. A caller without the manage permission (which the endpoint requires)
// passes enabled false so it never fires a request it knows will 403.
export const useFetchGrants = (resourceType: string, resourceId: string, enabled = true) =>
  useQuery({
    queryKey: [getGrantsKey, resourceType, resourceId],
    queryFn: async () =>
      (
        await api.get<{ grants: PermissionGrant[] }>(
          `/api/permissions?resource_type=${resourceType}&resource_id=${resourceId}`,
        )
      ).data.grants,
    enabled: enabled && resourceId !== "",
  });

// A person's answer for every permission the entity's rules may set, with its source; userId empty is the viewer.
export const useFetchExplanation = (resourceType: string, resourceId: string, userId = "", enabled = true) =>
  useQuery({
    queryKey: [getExplanationKey, resourceType, resourceId, userId],
    queryFn: async () =>
      (
        await api.get<PermissionExplanation>("/api/permissions/explain", {
          params: { resource_type: resourceType, resource_id: resourceId, ...(userId ? { user_id: userId } : {}) },
        })
      ).data,
    enabled: enabled && resourceId !== "",
  });

// The one source of the permission vocabulary — every grid/checklist in the app renders this, never a hardcoded list.
export const useFetchPermissionCatalog = () =>
  useQuery({
    queryKey: [getPermissionCatalogKey],
    queryFn: async () =>
      (await api.get<{ permissions: PermissionInfo[] }>("/api/permissions/catalog")).data.permissions,
    staleTime: Infinity,
  });

// The project areas Project access is set in, from the catalog; empty until it arrives.
export const useProjectAreas = (): PermissionDomain[] => {
  const { data: catalog } = useFetchPermissionCatalog();
  return useMemo(() => domainsIn(catalog ?? [], [PermissionArea.Project]), [catalog]);
};

interface SaveRulesInput {
  resourceType: "doc" | "play";
  resourceId: string;
  changes: RuleChange[];
}

const ruleBody = (resourceType: string, resourceId: string, { userId, state, actions }: RuleChange) => {
  if (resourceType === "play") return { resource_type: "play", resource_ids: [resourceId], user_ids: [userId], actions, state };
  return { doc_ids: [resourceId], user_ids: [userId], actions, state };
};

const refreshRules = (client: QueryClient, resourceType: string, resourceId: string) =>
  Promise.all([
    client.invalidateQueries({ queryKey: [getGrantsKey, resourceType, resourceId] }),
    client.invalidateQueries({ queryKey: [getExplanationKey, resourceType, resourceId] }),
  ]);

export const useSaveRules = () => {
  const client = useQueryClient();
  return useMutation({
    mutationFn: async ({ resourceType, resourceId, changes }: SaveRulesInput) => {
      for (const change of changes) await api.put("/api/permissions", ruleBody(resourceType, resourceId, change));
    },
    onSuccess: async (_, { resourceType, resourceId, changes }) => {
      await refreshRules(client, resourceType, resourceId);
      if (resourceType === "doc") await client.invalidateQueries({ queryKey: [getDocsKey] });
      if (changes.length > 0) toast.success("Permissions saved");
    },
    onError: (error) => toast.error(errorMessage(error)),
  });
};

interface GrantPayload {
  resource_type: string;
  resource_id: string;
}

// A rule changed on a doc or play: an open panel on it rereads its rules and answers.
export const permissionRulesFollower: LiveFollower = {
  "access.grant.changed": ({ resource_type, resource_id }: GrantPayload, { client }) => refreshRules(client, resource_type, resource_id),
};
