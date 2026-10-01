// Mirrors internal/tenancy/model.go's Workspace wire shape, trimmed to the fields this app reads.
export interface Workspace {
  id: string;
  name: string;
  // Names the workspace in web URLs (/<slug>/tickets/WEB-1), and picks the workspace a ticket key resolves in.
  slug: string;
}

// The server's /me answer: an Owner's list is the whole grid; a Restricted member's project areas sit in projects.
export interface MyWorkspaceInfo {
  role_name: string;
  permissions: string[];
  restricted?: boolean;
  projects?: { project_id: string; actions: string[] }[];
}

// What the viewer holds somewhere in the workspace, for what stands for every project (a tab, a deep link's gate).
const workspaceWidePermissions = (info: MyWorkspaceInfo | undefined): string[] => {
  if (!info?.restricted) return info?.permissions ?? [];
  return [...info.permissions, ...(info.projects ?? []).flatMap((project) => project.actions)];
};

// What the viewer holds inside one project; a project they hold nothing on answers as any project would.
export const projectPermissions = (info: MyWorkspaceInfo | undefined, projectId: string | undefined): string[] => {
  if (!info?.restricted) return info?.permissions ?? [];
  const project = info.projects?.find((candidate) => candidate.project_id === projectId);
  if (!project) return workspaceWidePermissions(info);
  return [...info.permissions, ...project.actions];
};
