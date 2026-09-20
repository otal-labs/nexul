import { zodResolver } from "@hookform/resolvers/zod";
import { useQuery } from "@tanstack/react-query";
import { useForm } from "react-hook-form";
import { z } from "zod";

import { api } from "@/api/client";
import { ErrorDisplay } from "@/components/ErrorDisplay";
import { FormInput } from "@/components/FormInput";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { Button } from "@/components/ui/button";
import { getProjectsKey } from "@/hooks/ProjectHooks";
import { useFetchWorkspaces } from "@/hooks/WorkspaceHooks";
import type { Project } from "@/models/Project";
import { SaveProjectFormSchema } from "@/models/Project";
import type { Workspace } from "@/models/Workspace";
import { SaveWorkspaceFormSchema } from "@/models/Workspace";

// Every install seeds exactly one workspace at this fixed id (tenancy.defaultWorkspaceID).
const DEFAULT_WORKSPACE_ID = "workspace-default";

// Reuses the project/workspace form schemas' validation/messages instead of redeclaring them.
const SetupWorkspaceFormSchema = z.object({
  workspaceName: SaveWorkspaceFormSchema.shape.name,
  projectName: SaveProjectFormSchema.shape.name,
  projectPrefix: SaveProjectFormSchema.shape.prefix,
});

type SetupWorkspaceFormData = z.infer<typeof SetupWorkspaceFormSchema>;

// Hands back ids alongside the validated values because the rename/prefix mutations can't run in this step.
export interface WorkspaceSetupData {
  workspaceId: string;
  workspaceName: string;
  projectId: string;
  projectName: string;
  projectPrefix: string;
}

interface SetupWorkspaceStepProps {
  onContinue: (data: WorkspaceSetupData) => void;
}

// GET /api/workspaces always comes back empty pre-wizard, so pick the default workspace by its seeded id.
const pickDefaultWorkspace = (workspaces: Workspace[] | undefined) =>
  workspaces?.find((w) => w.id === DEFAULT_WORKSPACE_ID) ?? workspaces?.[0];

// The default project still carries the empty prefix SetPrefix backfills once (ADR 0004); else the first.
const pickDefaultProject = (projects: Project[] | undefined) => projects?.find((p) => p.prefix === "") ?? projects?.[0];

// Every workspace ships a seeded default project with an empty prefix; this screen captures its name/prefix.
export const SetupWorkspaceStep = ({ onContinue }: SetupWorkspaceStepProps) => {
  const { data: workspaces } = useFetchWorkspaces();
  const currentWorkspace = pickDefaultWorkspace(workspaces);

  // GET /api/projects has no membership check, so this works pre-membership, unlike workspaces above.
  const { data: projects, isPending, error } = useQuery({
    queryKey: [getProjectsKey, DEFAULT_WORKSPACE_ID],
    queryFn: async () =>
      (await api.get<Project[]>("/api/projects", { params: { workspace_id: DEFAULT_WORKSPACE_ID } })).data,
  });
  const defaultProject = pickDefaultProject(projects);

  return (
    <div className="space-y-6">
      {isPending && <LoadingDisplay />}
      {error && <ErrorDisplay error={error} />}
      {projects && projects.length === 0 && <ErrorDisplay error={new Error("No project found")} />}
      {defaultProject && (
        // "Default" mirrors migration 0096's seeded name; an empty fallback fails min-length validation.
        <SetupWorkspaceForm
          workspaceName={currentWorkspace?.name ?? "Default"}
          project={defaultProject}
          onContinue={onContinue}
        />
      )}
    </div>
  );
};

interface SetupWorkspaceFormProps {
  workspaceName: string;
  project: Project;
  onContinue: (data: WorkspaceSetupData) => void;
}

// Split out so useForm's defaultValues (set once on mount) read the already-fetched workspace/project.
const SetupWorkspaceForm = ({ workspaceName, project, onContinue }: SetupWorkspaceFormProps) => {
  const form = useForm<SetupWorkspaceFormData>({
    defaultValues: {
      workspaceName,
      projectName: project.name,
      projectPrefix: project.prefix,
    },
    resolver: zodResolver(SetupWorkspaceFormSchema),
  });

  // Rename/SetPrefix need can_create_workspace (granted after this step), so hand values to the wizard shell.
  const onSubmit = (data: SetupWorkspaceFormData) => {
    onContinue({
      workspaceId: DEFAULT_WORKSPACE_ID,
      workspaceName: data.workspaceName,
      projectId: project.id,
      projectName: data.projectName,
      projectPrefix: data.projectPrefix,
    });
  };

  return (
    <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-6">
      <FormInput control={form.control} name="workspaceName" label="Workspace name" placeholder="Acme" />
      <div className="space-y-4">
        <h2 className="text-sm font-semibold text-muted-foreground">Default project</h2>
        <FormInput control={form.control} name="projectName" label="Project name" placeholder="General" />
        <FormInput control={form.control} name="projectPrefix" label="Project prefix" placeholder="GEN" maxLength={5} />
      </div>
      <Button type="submit" className="w-full" disabled={form.formState.isSubmitting}>
        {form.formState.isSubmitting ? "Saving…" : "Continue"}
      </Button>
    </form>
  );
};
