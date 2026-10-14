import { useQueries } from "@tanstack/react-query";

import { personLabel, type Person } from "@nexul/client-core/person";

import { useFetchMe } from "@/hooks/AuthHooks";
import { useFetchProjectPeople, useFetchProjectRoleIds, useFetchWorkspacePeople, workspacePeopleQuery } from "@/hooks/PeopleHooks";
import { useFetchGrants } from "@/hooks/PermissionHooks";
import { useFetchProject } from "@/hooks/ProjectHooks";
import { useFetchWorkspaceRoles } from "@/hooks/RoleHooks";
import { useFetchWorkspaces } from "@/hooks/WorkspaceHooks";
import { useFetchTeam } from "@/hooks/TeamHooks";
import { grantKey, parseTarget, targetKey, type TargetKind } from "@/models/PermissionPanel";
import type { Role } from "@/models/Role";

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
  // Set for a doc or folder: a target outside this project can't be listed until it is in it.
  projectId?: string | undefined;
  // A computer's: no roles, and every person who shares a workspace with the viewer, the viewer left out.
  everyWorkspace?: boolean;
  // An instance area: every workspace's roles, each named with its workspace, and every account.
  instance?: boolean | undefined;
  // The entity whose rules name targets: a role or person the lists above can't resolve is named as the rules list does.
  rulesOn?: { type: string; id: string } | undefined;
}

// Everyone in any workspace the viewer is in but the viewer, once, while everyWorkspace; undefined until all have loaded.
const usePeopleAround = (everyWorkspace: boolean): Person[] | undefined => {
  const { data: workspaces } = useFetchWorkspaces(everyWorkspace);
  const { data: me } = useFetchMe();
  return useQueries({
    queries: (everyWorkspace ? (workspaces ?? []) : []).map((w) => workspacePeopleQuery(w.id)),
    combine: (results) => {
      if (!everyWorkspace || !workspaces || !me || results.some((r) => !r.data)) return undefined;
      const people = new Map(results.flatMap((r) => r.data ?? []).map((p) => [p.user_id, p]));
      people.delete(me.user.id);
      return [...people.values()].sort((a, b) => personLabel(a).localeCompare(personLabel(b)));
    },
  });
};

const noRoles: Role[] = [];

// Every role and member of the workspace as targets, and why one can't be listed here yet.
export const usePanelTargets = ({ workspaceId, projectId, everyWorkspace = false, instance = false, rulesOn }: TargetScope) => {
  const { data: team } = useFetchTeam(instance);
  // Team is read for other reasons too, so it is only consulted for an area.
  const areaTeam = instance ? team : undefined;
  const { data: workspaceRoles } = useFetchWorkspaceRoles(everyWorkspace || instance ? "" : workspaceId);
  const { data: workspacePeople } = useFetchWorkspacePeople(everyWorkspace || instance ? undefined : workspaceId);
  const around = usePeopleAround(everyWorkspace);
  const teamRoles = areaTeam?.workspaces.flatMap((w) => w.roles.map((r) => ({ id: r.id, name: `${r.name} in ${w.name}`, is_owner_role: r.is_owner })));
  const teamPeople = areaTeam?.people.filter((p) => p.status !== "removed").map((p) => ({ user_id: p.id, login: p.login, display_name: p.display_name || p.name, avatar_url: p.avatar_url }));
  const byScope = <T,>(computer: T, area: T, workspace: T): T => {
    if (everyWorkspace) return computer;
    if (instance) return area;
    return workspace;
  };
  const roles = byScope<Pick<Role, "id" | "name" | "is_owner_role">[] | undefined>(around && noRoles, teamRoles, workspaceRoles);
  const people = byScope<Person[] | undefined>(around, teamPeople, workspacePeople);
  const { data: inProject } = useFetchProjectPeople(projectId);
  const { data: rolesIn } = useFetchProjectRoleIds(projectId);
  const { data: project } = useFetchProject(projectId);
  const { data: rules } = useFetchGrants(rulesOn?.type ?? "", rulesOn?.id ?? "", !!rulesOn);

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
    const named = rules?.find((g) => grantKey(g) === key)?.target_name ?? "";
    if (kind === "role") return { key, kind, id, name: named || "Unknown role", login: "", avatarUrl: "", known: false };
    return { key, kind, id, name: named || "Someone", login: named, avatarUrl: "", known: false };
  };

  const inside = (t: PanelTarget) => (t.kind === "role" ? rolesIn?.includes(t.id) : inProject?.some((p) => p.user_id === t.id));
  const blockedFor = (t: PanelTarget) => (projectId && inside(t) === false ? `Add ${t.name} to ${project?.name ?? "the project"} first` : undefined);

  return { roles: roleTargets, people: personTargets, ready: !!roles && !!people, targetOf, blockedFor, project };
};
