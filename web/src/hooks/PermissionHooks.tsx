import { type QueryClient, useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";

import { api, errorMessage } from "@/api/client";
import { getDocFoldersKey } from "@/hooks/DocFolderHooks";
import { getDocsKey } from "@/hooks/DocHooks";
import { getProjectPeopleKey } from "@/hooks/PeopleHooks";
import {
  type PermissionExplanation,
  type PermissionGrant,
  type PermissionInfo,
  type RuleChange,
} from "@/models/Permission";
import { parseTarget, type PanelEntity } from "@/models/PermissionPanel";
import type { LiveFollower } from "@/lib/live";

export const getGrantsKey = "getGrants";
export const getExplanationKey = "getExplanation";
const getPermissionCatalogKey = "getPermissionCatalog";

// A doc's, folder's, play's or project's rules, one row per role or person. A caller without the manage permission (which the endpoint requires)
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

// The explain request's target: "role:<id>" asks for a role, any other non-empty value names a person.
const explainTarget = (target: string) => {
  if (target.startsWith("role:")) return { role_id: target.slice("role:".length) };
  return target ? { user_id: target } : {};
};

// A person's or role's answer for every permission the entity's rules may set, with its source; target empty is the
// viewer, a person by user id, and "role:<id>" a role.
export const useFetchExplanation = (resourceType: string, resourceId: string, target = "", enabled = true) =>
  useQuery({
    queryKey: [getExplanationKey, resourceType, resourceId, target],
    queryFn: async () =>
      (
        await api.get<PermissionExplanation>("/api/permissions/explain", {
          params: { resource_type: resourceType, resource_id: resourceId, ...explainTarget(target) },
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

interface SaveRulesInput {
  resourceType: PanelEntity["type"];
  resourceId: string;
  changes: RuleChange[];
  // The toast once it lands, when the change says more than "Permissions saved".
  done?: string;
}

const ruleBody = (resourceType: string, resourceId: string, { target, state, actions }: RuleChange) => {
  const { kind, id } = parseTarget(target);
  const to = kind === "role" ? { role_ids: [id] } : { user_ids: [id] };
  if (resourceType === "doc") return { doc_ids: [resourceId], ...to, actions, state };
  return { resource_type: resourceType, resource_ids: [resourceId], ...to, actions, state };
};

const refreshRules = (client: QueryClient, resourceType: string, resourceId: string) =>
  Promise.all([
    client.invalidateQueries({ queryKey: [getGrantsKey, resourceType, resourceId] }),
    client.invalidateQueries({ queryKey: [getExplanationKey, resourceType, resourceId] }),
    resourceType === "project" && client.invalidateQueries({ queryKey: [getProjectPeopleKey, resourceId] }),
  ]);

export const useSaveRules = () => {
  const client = useQueryClient();
  return useMutation({
    mutationFn: async ({ resourceType, resourceId, changes }: SaveRulesInput) => {
      for (const change of changes) await api.put("/api/permissions", ruleBody(resourceType, resourceId, change));
    },
    onSuccess: async (_, { resourceType, resourceId, changes, done }) => {
      await refreshRules(client, resourceType, resourceId);
      if (resourceType === "doc" || resourceType === "folder") await client.invalidateQueries({ queryKey: [getDocsKey] });
      if (resourceType === "folder") await client.invalidateQueries({ queryKey: [getDocFoldersKey] });
      if (changes.length > 0) toast.success(done ?? "Permissions saved");
    },
    onError: (error) => toast.error(errorMessage(error)),
  });
};

interface GrantPayload {
  resource_type: string;
  resource_id: string;
}

// A rule changed on a doc, folder, play or project: an open panel or people card on it rereads its rules and answers.
export const permissionRulesFollower: LiveFollower = {
  "access.grant.changed": ({ resource_type, resource_id }: GrantPayload, { client }) => refreshRules(client, resource_type, resource_id),
};
