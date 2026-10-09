// The permission each gated area needs: the web's sidebar and route guard and the phone's tabs and screens read it.
export const AREA_PERMISSION = {
  runners: "runners:read",
  topology: "topology:read",
  automations: "automations:read",
  projects: "projects:read",
  tickets: "tickets:read",
  memories: "memories:read",
  docs: "docs:read",
  stacks: "stacks:read",
  stackLogs: "stacks:logs",
  deploys: "deploys:read",
  dns: "dns:read",
  newConversation: "chat:write",
  editChannels: "channels:write",
  deleteChannels: "channels:delete",
  bots: "botwebhook:read",
  editBots: "botwebhook:write",
  deleteBots: "botwebhook:delete",
  newProject: "projects:write",
  newDoc: "docs:write",
  editNotes: "tickets:write",
  editTickets: "tickets:write",
} as const;

export type Area = keyof typeof AREA_PERMISSION;

// Mirrors internal/tenancy/handler.go's meResponse; a Restricted member's permissions carry no project area, so
// projects lists what they hold in each project they may open.
export interface MyWorkspaceInfo {
  role_name: string;
  permissions: string[];
  restricted?: boolean;
  projects?: { project_id: string; actions: string[] }[];
}

// The one place client-side permission logic lives: a thin wrapper the server resolves, the client just reads.
export const hasPermission = (permissions: readonly string[] | undefined, value: string): boolean =>
  (permissions ?? []).includes(value);

// What the viewer holds somewhere in the workspace, for what stands for every project (a nav entry, a deep link's gate).
export const workspaceWidePermissions = (info: MyWorkspaceInfo | undefined): string[] => {
  if (!info?.restricted) return info?.permissions ?? [];
  return [...info.permissions, ...(info.projects ?? []).flatMap((project) => project.actions)];
};

// What the viewer holds inside one project: the workspace areas plus, for a Restricted member, that project's access.
// A project they hold nothing on (none in view yet, or one just taken away) answers as any project would.
export const projectPermissions = (info: MyWorkspaceInfo | undefined, projectId: string | undefined): string[] => {
  if (!info?.restricted) return info?.permissions ?? [];
  const project = info.projects?.find((candidate) => candidate.project_id === projectId);
  if (!project) return workspaceWidePermissions(info);
  return [...info.permissions, ...project.actions];
};
