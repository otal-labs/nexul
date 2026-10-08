import { zodResolver } from "@hookform/resolvers/zod";
import { useForm } from "react-hook-form";

import { ErrorDisplay } from "@/components/ErrorDisplay";
import { FormInput } from "@/components/FormInput";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { Button } from "@/components/ui/button";
import { useFetchWorkspaces } from "@/hooks/WorkspaceHooks";
import type { Workspace } from "@/models/Workspace";
import { SaveWorkspaceFormSchema, type SaveWorkspaceFormData } from "@/models/Workspace";

// Every install seeds exactly one workspace at this fixed id (tenancy.DefaultWorkspaceID).
const DEFAULT_WORKSPACE_ID = "workspace-default";

// Hands back the id alongside the name because the rename can't run in this step.
export interface WorkspaceSetupData {
  workspaceId: string;
  workspaceName: string;
}

interface SetupWorkspaceStepProps {
  onContinue: (data: WorkspaceSetupData) => Promise<void>;
}

// GET /api/workspaces comes back empty before the wizard makes the owner a member, so the seeded name is the fallback.
const defaultWorkspaceName = (workspaces: Workspace[]) =>
  workspaces.find((w) => w.id === DEFAULT_WORKSPACE_ID)?.name ?? "Default";

// Names the workspace only: projects come from the project wizard, which the owner lands in next.
export const SetupWorkspaceStep = ({ onContinue }: SetupWorkspaceStepProps) => {
  const { data: workspaces, isPending, error } = useFetchWorkspaces();

  return (
    <div className="space-y-6">
      {isPending && <LoadingDisplay />}
      {error && <ErrorDisplay error={error} />}
      {workspaces && <SetupWorkspaceForm workspaceName={defaultWorkspaceName(workspaces)} onContinue={onContinue} />}
    </div>
  );
};

interface SetupWorkspaceFormProps {
  workspaceName: string;
  onContinue: (data: WorkspaceSetupData) => Promise<void>;
}

// Split out so useForm's defaultValues (set once on mount) read the already-fetched name.
const SetupWorkspaceForm = ({ workspaceName, onContinue }: SetupWorkspaceFormProps) => {
  const form = useForm<SaveWorkspaceFormData>({
    defaultValues: { name: workspaceName },
    resolver: zodResolver(SaveWorkspaceFormSchema),
  });

  // Renaming needs workspaces:write, which the owner holds only once the wizard shell makes them owner, so it renames.
  const onSubmit = (data: SaveWorkspaceFormData) =>
    onContinue({ workspaceId: DEFAULT_WORKSPACE_ID, workspaceName: data.name });

  return (
    <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-6">
      <FormInput control={form.control} name="name" label="Workspace name" placeholder="Acme" autoFocus />
      <Button type="submit" className="w-full" loading={form.formState.isSubmitting}>
        Continue
      </Button>
    </form>
  );
};
