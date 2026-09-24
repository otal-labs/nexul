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
  /** Immutable 2-5 uppercase-letter tag, unique per workspace, for the human-readable ticket id (PREFIX-N). */
  prefix: string;
  position: number;
  /** Owner-configured lucide-react icon name, "" when unset — falls back to prefix-only rendering. */
  icon: string;
  /** Where the project's tests live, as answered in the project wizard; the interview starts from it. */
  tests_location: TestsLocation;
  created_at: string;
  updated_at: string;
}

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

export const projectSettingsPath = (token: string): string => `/projects/${token}/settings`;

export const docPath = (token: string, docId: string): string => `/docs/${token}/${docId}`;

// token "" is a workspace-scoped memory: the bare, project-less route.
export const memoryPath = (token: string, memoryId: string): string =>
  token !== "" ? `/memories/${token}/${memoryId}` : `/memories/${memoryId}`;

export interface RepoRef {
  owner: string;
  name: string;
  full_name: string;
  /** Which connected git connector hosts this repo; "github" is the only real option today. */
  connector_id: string;
  role: RepoRole;
}

export interface DeleteImpact {
  tickets: number;
  repos: number;
  services: number;
}

export const SaveProjectFormSchema = z.object({
  name: z.string().trim().min(1, "Project name is required"),
  prefix: z
    .string()
    .trim()
    .regex(/^[A-Za-z]{2,5}$/, "Prefix must be 2-5 letters"),
  icon: z.enum(["", ...PROJECT_ICON_NAMES]),
});

export type SaveProjectFormData = z.infer<typeof SaveProjectFormSchema>;

export const AddProjectRepoFormSchema = z.object({
  owner: z.string().trim().min(1, "Owner is required"),
  name: z.string().trim().min(1, "Repo name is required"),
  connectorId: z.string().trim().min(1, "Connector is required"),
});

export type AddProjectRepoFormData = z.infer<typeof AddProjectRepoFormSchema>;
