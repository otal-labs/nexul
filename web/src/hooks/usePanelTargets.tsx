import { personLabel, unknownPerson } from "@nexul/client-core/person";

import { useFetchProjectPeople, useFetchProjectRoleIds, useFetchWorkspacePeople } from "@/hooks/PeopleHooks";
import { useFetchProject } from "@/hooks/ProjectHooks";
import { useFetchWorkspaceRoles } from "@/hooks/RoleHooks";
import { parseTarget, targetKey, type TargetKind } from "@/models/PermissionPanel";

// A role or person a rule may name, as the panel and a project's people card show it.
export interface PanelTarget {
  key: string;
  kind: TargetKind;
  id: string;
  name: string;
  login: string;
  avatarUrl: string;
  // False for a role or account the workspace no longer lists.
  known: boolean;
}

interface TargetScope {
  workspaceId: string;
  // Set for a doc: a target outside this project can't be listed until it is in it.
  projectId?: string | undefined;
}

// Every role and member of the workspace as targets, and why one can't be listed here yet.
export const usePanelTargets = ({ workspaceId, projectId }: TargetScope) => {
  const { data: roles } = useFetchWorkspaceRoles(workspaceId);
  const { data: people } = useFetchWorkspacePeople(workspaceId);
  const { data: inProject } = useFetchProjectPeople(projectId);
  const { data: rolesIn } = useFetchProjectRoleIds(projectId);
  const { data: project } = useFetchProject(projectId);

  const allRoles: PanelTarget[] = (roles ?? []).map((r) => ({ key: targetKey("role", r.id), kind: "role", id: r.id, name: r.name, login: "", avatarUrl: "", known: true }));
  // The Owner role reaches everything whatever a rule says, so it is never offered as a target.
  const roleTargets = allRoles.filter((t) => !roles?.find((r) => r.id === t.id)?.is_owner_role);
  const personTargets: PanelTarget[] = (people ?? []).map((p) => ({
    key: targetKey("person", p.user_id), kind: "person", id: p.user_id, name: personLabel(p), login: p.login, avatarUrl: p.avatar_url, known: true,
  }));

  const targetOf = (key: string): PanelTarget => {
    const found = [...allRoles, ...personTargets].find((t) => t.key === key);
    if (found) return found;
    const { kind, id } = parseTarget(key);
    const stranger = unknownPerson(id);
    return { key, kind, id, name: kind === "role" ? "Unknown role" : personLabel(stranger), login: stranger.login, avatarUrl: "", known: false };
  };

  const inside = (t: PanelTarget) => (t.kind === "role" ? rolesIn?.includes(t.id) : inProject?.some((p) => p.user_id === t.id));
  const blockedFor = (t: PanelTarget) => (projectId && inside(t) === false ? `Add ${t.name} to ${project?.name ?? "the project"} first` : undefined);

  return { roles: roleTargets, people: personTargets, ready: !!roles && !!people, targetOf, blockedFor, project };
};
