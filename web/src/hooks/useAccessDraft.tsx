import { useShallow } from "zustand/react/shallow";

import { useAccessDraftStore, type ProjectListing } from "@/stores/accessDraftStore";
import type { AccessGrid, AccessGridSave } from "@/models/AccessGrid";

const sameSet = (a: string[], b: string[]) => a.length === b.length && a.every((id) => b.includes(id));

const apart = (a: string[], b: string[]) => a.filter((id) => !b.includes(id)).length + b.filter((id) => !a.includes(id)).length;

const flip = (ids: string[], id: string, on: boolean) => (on ? [...ids.filter((x) => x !== id), id] : ids.filter((x) => x !== id));

// The grid as ticked on screen: the held ticks over the server's rows, and the rows that differ from it.
export const useAccessDraft = (grid: AccessGrid) => {
  const draft = useAccessDraftStore(useShallow((s) => ({ projects: s.projects, channels: s.channels })));
  const { setProjects, setChannels, clearProjects, clearChannels } = useAccessDraftStore.getState();

  const listing = (projectId: string): ProjectListing => {
    const held = draft.projects[projectId];
    if (held) return held;
    const project = grid.projects.find((p) => p.id === projectId);
    return { roleIds: project?.role_ids ?? [], userIds: project?.user_ids ?? [] };
  };
  const hidden = (channelId: string): string[] =>
    draft.channels[channelId] ?? grid.channels.find((c) => c.id === channelId)?.hidden_role_ids ?? [];

  const projectChanges: NonNullable<AccessGridSave["projects"]> = grid.projects
    .filter((p) => {
      const held = draft.projects[p.id];
      return !!held && (!sameSet(held.roleIds, p.role_ids) || !sameSet(held.userIds, p.user_ids));
    })
    .map((p) => ({ project_id: p.id, role_ids: listing(p.id).roleIds, user_ids: listing(p.id).userIds }));
  const channelChanges: NonNullable<AccessGridSave["channels"]> = grid.channels
    .filter((c) => !!draft.channels[c.id] && !sameSet(hidden(c.id), c.hidden_role_ids))
    .map((c) => ({ channel_id: c.id, hidden_role_ids: hidden(c.id) }));

  // How many boxes differ from the server: each role or person added or taken off a project, each role a channel's hiding changed.
  const projectCells = projectChanges.reduce((n, change) => {
    const p = grid.projects.find((x) => x.id === change.project_id);
    return n + apart(change.role_ids, p?.role_ids ?? []) + apart(change.user_ids, p?.user_ids ?? []);
  }, 0);
  const channelCells = channelChanges.reduce(
    (n, change) => n + apart(change.hidden_role_ids, grid.channels.find((c) => c.id === change.channel_id)?.hidden_role_ids ?? []),
    0,
  );
  const serverProject = (projectId: string) => grid.projects.find((p) => p.id === projectId);
  const serverChannel = (channelId: string) => grid.channels.find((c) => c.id === channelId);

  return {
    listing,
    hidden,
    roleChanged: (projectId: string, roleId: string) =>
      listing(projectId).roleIds.includes(roleId) !== (serverProject(projectId)?.role_ids.includes(roleId) ?? false),
    hiddenChanged: (channelId: string, roleId: string) =>
      hidden(channelId).includes(roleId) !== (serverChannel(channelId)?.hidden_role_ids.includes(roleId) ?? false),
    projectChanges,
    channelChanges,
    projectCells,
    channelCells,
    toggleRole: (projectId: string, roleId: string) => {
      const now = listing(projectId);
      setProjects({ [projectId]: { ...now, roleIds: flip(now.roleIds, roleId, !now.roleIds.includes(roleId)) } });
    },
    togglePerson: (projectId: string, userId: string) => {
      const now = listing(projectId);
      setProjects({ [projectId]: { ...now, userIds: flip(now.userIds, userId, !now.userIds.includes(userId)) } });
    },
    setRoleOnProjects: (roleId: string, projectIds: string[], on: boolean) =>
      setProjects(Object.fromEntries(projectIds.map((id) => [id, { ...listing(id), roleIds: flip(listing(id).roleIds, roleId, on) }]))),
    toggleHidden: (channelId: string, roleId: string) =>
      setChannels({ [channelId]: flip(hidden(channelId), roleId, !hidden(channelId).includes(roleId)) }),
    setRoleOnChannels: (roleId: string, channelIds: string[], visible: boolean) =>
      setChannels(Object.fromEntries(channelIds.map((id) => [id, flip(hidden(id), roleId, !visible)]))),
    clearProjects,
    clearChannels,
  };
};
