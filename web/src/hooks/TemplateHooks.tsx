import { queryOptions, useMutation, useQueries, useQuery, useQueryClient, type QueryClient } from "@tanstack/react-query";
import { toast } from "sonner";

import { api, errorMessage } from "@/api/client";
import { getInterviewTemplateKey } from "@/hooks/MemoryHooks";
import { getApplicablePlaysKey, getWorkspacePlaysKey } from "@/hooks/PlayHooks";
import { getProjectTicketTypesKey, getTicketTypesKey } from "@/hooks/TicketTypeHooks";
import { getWorkspacesKey, myRoleQuery, useFetchWorkspaces } from "@/hooks/WorkspaceHooks";
import { hasPermission } from "@/models/Permission";
import type { Template, TemplateKind, TemplateLocation } from "@/models/Template";
import type { Workspace } from "@/models/Workspace";
import type { LiveFollower } from "@/lib/live";

export const getTemplatesKey = "getTemplates";
export const getTemplateKey = "getTemplate";

// Every query that shows a template's text at some layer; an instance change or a clone can move any of them.
export const TEMPLATE_QUERY_KEYS = [
  getTemplatesKey,
  getTemplateKey,
  getWorkspacesKey,
  getInterviewTemplateKey,
  getWorkspacePlaysKey,
  getApplicablePlaysKey,
  getTicketTypesKey,
  getProjectTicketTypesKey,
];

const refreshTemplates = (client: QueryClient) =>
  Promise.all(TEMPLATE_QUERY_KEYS.map((key) => client.invalidateQueries({ queryKey: [key] })));

// Open to every signed-in member: the instance templates are what every workspace starts from.
export const useFetchTemplates = () =>
  useQuery({
    queryKey: [getTemplatesKey],
    queryFn: async () => (await api.get<Template[]>("/api/templates")).data,
  });

// One template at any location; the clone dialog reads the target's before overwriting it.
export const templateQuery = (kind: TemplateKind, key: string, at: TemplateLocation) =>
  queryOptions({
    queryKey: [getTemplateKey, kind, key, at],
    queryFn: async () => (await api.get<Template>(`/api/templates/${kind}`, { params: { key, ...at } })).data,
  });

export const useSaveInstanceTemplate = () => {
  const client = useQueryClient();
  return useMutation({
    mutationFn: async ({ kind, key, body }: { kind: TemplateKind; key: string; body: string }) =>
      (await api.put<Template>(`/api/templates/${kind}`, { key, body })).data,
    onSuccess: async () => {
      await refreshTemplates(client);
      toast.success("Template saved");
    },
    onError: (error) => toast.error(errorMessage(error)),
  });
};

export const useResetInstanceTemplate = () => {
  const client = useQueryClient();
  return useMutation({
    mutationFn: async ({ kind, key }: { kind: TemplateKind; key: string }) =>
      (await api.delete<Template>(`/api/templates/${kind}`, { params: { key } })).data,
    onSuccess: async () => {
      await refreshTemplates(client);
      toast.success("Template reset to the default");
    },
    onError: (error) => toast.error(errorMessage(error)),
  });
};

// Resets a workspace's or project's template to the instance's text.
export const useResetTemplate = () => {
  const client = useQueryClient();
  return useMutation({
    mutationFn: async ({ kind, key, at }: { kind: TemplateKind; key: string; at: TemplateLocation }) =>
      (await api.post<Template>("/api/templates/reset", { kind, key, at })).data,
    onSuccess: async () => {
      await refreshTemplates(client);
      toast.success("Reset to the instance template");
    },
    onError: (error) => toast.error(errorMessage(error)),
  });
};

// No error toast: the clone dialog shows the server's message (a missing play or ticket type) inline.
export const useCloneTemplate = () => {
  const client = useQueryClient();
  return useMutation({
    mutationFn: async (input: { kind: TemplateKind; key: string; from: TemplateLocation; to: TemplateLocation }) =>
      (await api.post<Template>("/api/templates/clone", input)).data,
    onSuccess: async () => {
      await refreshTemplates(client);
      toast.success("Template cloned");
    },
  });
};

// The viewer's workspaces where they hold permission; undefined until every role has answered.
export const useEditableWorkspaces = (permission: string): Workspace[] | undefined => {
  const { data: workspaces } = useFetchWorkspaces();
  const roles = useQueries({ queries: (workspaces ?? []).map((w) => myRoleQuery(w.id)) });
  if (!workspaces || roles.some((r) => r.isPending)) return undefined;
  return workspaces.filter((_, i) => hasPermission(roles[i]?.data?.permissions, permission));
};

// An instance template changes what every unedited workspace shows and what each copy is compared with.
export const templateFollower: LiveFollower = {
  "instance_template.updated": (_payload: unknown, { client }) => refreshTemplates(client),
};
