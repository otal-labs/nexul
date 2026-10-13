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
  autoPlays: "autoplays:read",
  editAutoPlays: "autoplays:write",
  deleteAutoPlays: "autoplays:delete",
  newProject: "projects:create",
  editProjects: "projects:write",
  newDoc: "docs:write",
  editNotes: "tickets:write",
  editTickets: "tickets:write",
} as const;

export type Area = keyof typeof AREA_PERMISSION;

// One role a member holds, as the server names it.
export interface HeldRole {
  id: string;
  name: string;
  is_owner: boolean;
}

// Mirrors internal/tenancy/handler.go's meResponse. Projects are opt-in (ADR 0148): a non-Owner's permissions carry no
// project area, and projects lists what they hold in each project they are in.
export interface MyWorkspaceInfo {
  role_name: string;
  // Every role held, in the order given; role_name names them joined.
  roles?: HeldRole[];
  permissions: string[];
  projects?: { project_id: string; actions: string[] }[];
}

// The one place client-side permission logic lives: a thin wrapper the server resolves, the client just reads.
export const hasPermission = (permissions: readonly string[] | undefined, value: string): boolean =>
  (permissions ?? []).includes(value);

// What the viewer holds somewhere in the workspace, for what stands for every project (a nav entry, a deep link's gate).
export const workspaceWidePermissions = (info: MyWorkspaceInfo | undefined): string[] => [
  ...(info?.permissions ?? []),
  ...(info?.projects ?? []).flatMap((project) => project.actions),
];

// What the viewer holds inside one project: the workspace areas plus what they hold in that project. A project not
// listed (none in view yet, or one just taken away) answers as any project would.
export const projectPermissions = (info: MyWorkspaceInfo | undefined, projectId: string | undefined): string[] => {
  const project = info?.projects?.find((candidate) => candidate.project_id === projectId);
  if (!project) return workspaceWidePermissions(info);
  return [...(info?.permissions ?? []), ...project.actions];
};
