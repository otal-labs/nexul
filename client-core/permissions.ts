// The permission each gated area needs: the web's sidebar and route guard and the phone's tabs and screens read it.
export const AREA_PERMISSION = {
  topology: "topology:read",
  automations: "automations:read",
  plays: "plays:read",
  projects: "projects:read",
  tickets: "tickets:read",
  memories: "memories:read",
  docs: "docs:read",
  stacks: "stacks:read",
  editStacks: "stacks:write",
  stackLogs: "stacks:logs",
  deploys: "deploys:read",
  editDeploys: "deploys:write",
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
  editDocs: "docs:write",
  editNotes: "tickets:write",
  editTickets: "tickets:write",
  deleteTickets: "tickets:delete",
  newMeeting: "calendar:write",
  changeAnyMeeting: "calendar:delete",
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
  // What a non-Owner who may create projects holds in one they create; a server from before it sends none.
  new_project_actions?: string[];
  // False until an Owner finishes the workspace's one-time access setup; a server from before it sends none.
  access_setup_finished?: boolean;
  // True when any role the viewer holds shows an action they can't take dimmed with the reason (ADR 0152); a server from before it sends none.
  show_locked_actions?: boolean;
}

// The one place client-side permission logic lives: a thin wrapper the server resolves, the client just reads.
export const hasPermission = (permissions: readonly string[] | undefined, value: string): boolean =>
  (permissions ?? []).includes(value);

// What the viewer holds in the workspace or in some project they are in, for what the server checks against that.
export const heldPermissions = (info: MyWorkspaceInfo | undefined): string[] => [
  ...(info?.permissions ?? []),
  ...(info?.projects ?? []).flatMap((project) => project.actions),
];

// What stands for every project (a nav entry, a deep link's gate, the project wizard's door): held, or held in one they create.
export const workspaceWidePermissions = (info: MyWorkspaceInfo | undefined): string[] => [
  ...heldPermissions(info),
  ...(info?.new_project_actions ?? []),
];

// What the viewer holds inside one project: the workspace areas plus what they hold in that project. With none in view it
// answers as any project would; a project not listed (one just taken away) answers from what they hold elsewhere.
export const projectPermissions = (info: MyWorkspaceInfo | undefined, projectId: string | undefined): string[] => {
  if (!projectId) return workspaceWidePermissions(info);
  const project = info?.projects?.find((candidate) => candidate.project_id === projectId);
  if (!project) return heldPermissions(info);
  return [...(info?.permissions ?? []), ...project.actions];
};
