import { useShallow } from "zustand/react/shallow";

import { useAccessDraftStore, type ProjectListing } from "@/stores/accessDraftStore";
import type { AccessGrid, AccessGridSave } from "@/models/AccessGrid";

const sameSet = (a: string[], b: string[]) => a.length === b.length && a.every((id) => b.includes(id));

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

  return {
    listing,
    hidden,
    projectChanges,
    channelChanges,
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
