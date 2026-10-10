import { z } from "zod";

import type { RepoRole, TestsLocation } from "@/enums/Project";

// Values are lucide-react component names stored verbatim, looked up directly wherever an icon renders.
export const PROJECT_ICON_NAMES = [
  "Box",
  "Rocket",
  "Server",
  "Globe",
  "Database",
  "Layers",
  "Terminal",
  "Shield",
  "Zap",
  "Package",
  "Cpu",
  "Cloud",
] as const;

export type ProjectIconName = (typeof PROJECT_ICON_NAMES)[number];

export interface Project {
  id: string;
  name: string;
  /** Immutable 2-5 character uppercase tag (a letter, then letters or digits), unique per workspace, for the human-readable ticket id (PREFIX-N). */
  prefix: string;
  position: number;
  workspace_id: string;
  /** Owner-configured lucide-react icon name, "" when unset — falls back to prefix-only rendering. */
  icon: string;
  /** Where the project's tests live, as answered in the project wizard; the interview starts from it. */
  tests_location: TestsLocation;
  setup: ProjectSetup;
  created_at: string;
  updated_at: string;
}

export type SetupMark = "done" | "skipped";

// The project wizard's record of a project (ADR 0143): until finished, the sidebar offers Continue setup.
export interface ProjectSetup {
  stack_id?: string;
  env_keys?: string[];
  finished: boolean;
  steps: Partial<Record<string, SetupMark>>;
}

export const inSetup = (project: Project): boolean => !project.setup.finished;

// The letters a project shows on its tile: the prefix, or the name's first letter before one was backfilled.
export const projectTile = (project: Project): string => project.prefix || project.name[0]?.toUpperCase() || "";

// A project's URL token: its prefix, so links survive renames, or the id if a prefix was never backfilled.
export const projectToken = (project: Project): string => project.prefix || project.id;

// Resolves a token when only the id is at hand; returns the raw id if the list hasn't loaded the project.
export const projectTokenById = (projects: Project[], projectId: string): string => {
  const project = projects.find((p) => p.id === projectId);
  return project ? projectToken(project) : projectId;
};

export const resolveProject = (projects: Project[], param: string): Project | undefined =>
  projects.find((p) => p.prefix.toLowerCase() === param.toLowerCase()) ??
  projects.find((p) => p.id === param);

export const boardPath = (project: Project): string => `/board/${projectToken(project)}`;

// The token of the project a URL is scoped to; undefined on pages that belong to no one project.
const projectScopedPaths = [/^\/board\/([^/]+)/, /^\/projects\/([^/]+)\//, /^\/(?:docs|memories)\/([^/]+)\/[^/]+/];

export const projectTokenFromPath = (pathname: string): string | undefined => {
  for (const pattern of projectScopedPaths) {
    const token = pattern.exec(pathname)?.[1];
    if (token) return token;
  }
  return undefined;
};

// A ticket page names its project by the key's prefix ("/tickets/WEB-12").
const ticketProjectPath = /^\/tickets\/([A-Za-z][A-Za-z0-9]{1,4})-\d+(?:\/|$)/;

// The token of the project a page shows, ticket pages included; undefined on pages that belong to no one project.
export const openProjectToken = (pathname: string): string | undefined =>
  projectTokenFromPath(pathname) ?? ticketProjectPath.exec(pathname)?.[1];

// Switching project keeps you on its settings or interview; anywhere else lands on the new project's board.
export const switchProjectPath = (pathname: string, project: Project): string => {
  const page = /^\/projects\/[^/]+\/(settings|interview)/.exec(pathname)?.[1];
  if (page) return `/projects/${projectToken(project)}/${page}`;
  return boardPath(project);
};

export const projectSettingsPath = (token: string): string => `/projects/${token}/settings`;

// Every way to create a project opens the project wizard; it is the only thing that makes one.
export const NEW_PROJECT_PATH = "/wizard/project/project";

export const interviewPath = (token: string): string => `/projects/${token}/interview`;

export const docPath = (token: string, docId: string): string => `/docs/${token}/${docId}`;

export const memoryPath = (token: string, memoryId: string): string => `/memories/${token}/${memoryId}`;

export interface RepoRef {
  owner: string;
  name: string;
  full_name: string;
  /** Which connected git connector hosts this repo; "github" is the only real option today. */
  connector_id: string;
  role: RepoRole;
}

// A Restricted member holding access to a project, named the way People shows them.
export interface RestrictedMember {
  user_id: string;
  name: string;
}

export interface DeleteImpact {
  tickets: number;
  repos: number;
  services: number;
  restricted_members?: RestrictedMember[];
}

export interface ProjectAccessEntry extends RestrictedMember {
  actions: string[];
}

export const SaveProjectFormSchema = z.object({
  name: z.string().trim().min(1, "Project name is required"),
  prefix: z
    .string()
    .trim()
    .regex(/^[A-Za-z][A-Za-z0-9]{1,4}$/, "Prefix is 2–5 letters or digits, starting with a letter"),
  icon: z.enum(["", ...PROJECT_ICON_NAMES]),
});

export type SaveProjectFormData = z.infer<typeof SaveProjectFormSchema>;

export const AddProjectRepoFormSchema = z.object({
  owner: z.string().trim().min(1, "Owner is required"),
  name: z.string().trim().min(1, "Repo name is required"),
  connectorId: z.string().trim().min(1, "Connector is required"),
});

export type AddProjectRepoFormData = z.infer<typeof AddProjectRepoFormSchema>;
