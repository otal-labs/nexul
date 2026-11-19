import { Controller, useFormContext, useFormState, useWatch } from "react-hook-form";

import { Button } from "@/components/ui/button";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { RolesChecklist } from "@/components/role/RolesChecklist";
import { useFetchWorkspaces } from "@/hooks/WorkspaceHooks";
import type { CreateInvitationFormData } from "@/models/Invitation";

interface InvitationGrantRowProps {
  index: number;
  remove: (index: number) => void;
  // Off for the only workspace: a link that admits nowhere gives nothing.
  removable: boolean;
}

// One workspace an invitation admits to: the workspace, then the roles the link gives there, nothing ticked at first.
export const InvitationGrantRow = ({ index, remove, removable }: InvitationGrantRowProps) => {
  const { control, setValue } = useFormContext<CreateInvitationFormData>();
  const { errors } = useFormState({ control });
  const workspaceId = useWatch({ control, name: `grants.${index}.workspace_id` });
  const { data: workspaces } = useFetchWorkspaces();
  const workspaceName = workspaces?.find((workspace) => workspace.id === workspaceId)?.name ?? "";
  const grantErrors = errors.grants?.[index];

  return (
    <li className="space-y-3">
      <div className="flex min-h-11 flex-wrap items-center gap-2 border-b border-border pb-3">
        <Controller
          control={control}
          name={`grants.${index}.workspace_id`}
          render={({ field }) => (
            <Select
              value={field.value}
              onValueChange={(value) => {
                field.onChange(value);
                setValue(`grants.${index}.role_ids`, []);
              }}
            >
              <SelectTrigger aria-label={`Workspace ${index + 1}`} className="h-8 min-w-0 flex-1">
                <SelectValue placeholder="Choose a workspace" />
              </SelectTrigger>
              <SelectContent>
                {(workspaces ?? []).map((workspace) => (
                  <SelectItem key={workspace.id} value={workspace.id}>{workspace.name}</SelectItem>
                ))}
              </SelectContent>
            </Select>
          )}
        />
        {removable && (
          <Button type="button" variant="ghost" size="sm" aria-label={`Remove workspace ${index + 1}`} onClick={() => remove(index)}>
            Remove
          </Button>
        )}
      </div>
      {grantErrors?.workspace_id?.message && <p role="alert" className="text-sm text-destructive">{grantErrors.workspace_id.message}</p>}
      {workspaceId && (
        <Controller
          control={control}
          name={`grants.${index}.role_ids`}
          render={({ field }) => (
            <RolesChecklist
              key={workspaceId}
              workspaceId={workspaceId}
              value={field.value}
              onChange={field.onChange}
              title={`Roles in ${workspaceName}`}
              emptyNote="Pick at least one role. An invitation gives a role on joining."
            />
          )}
        />
      )}
    </li>
  );
};
