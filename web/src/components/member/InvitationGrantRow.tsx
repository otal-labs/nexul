import { useEffect } from "react";
import { Controller, useFormContext, useFormState, useWatch } from "react-hook-form";

import { PermissionGrid } from "@/components/access/PermissionGrid";
import { PermissionLevels } from "@/components/access/PermissionLevels";
import { InvitationProjectAccess } from "@/components/member/InvitationProjectAccess";
import { Collapsible, CollapsibleContent, CollapsibleTrigger } from "@/components/ui/collapsible";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { useFetchPermissionCatalog } from "@/hooks/PermissionHooks";
import { useFetchWorkspaceRoles } from "@/hooks/RoleHooks";
import { useFetchWorkspaces } from "@/hooks/WorkspaceHooks";
import type { CreateInvitationFormData } from "@/models/Invitation";

interface InvitationGrantRowProps {
  index: number;
  remove: (index: number) => void;
}

// One workspace an invitation admits to: the workspace and role, its Every project block, then optional overrides.
export const InvitationGrantRow = ({ index, remove }: InvitationGrantRowProps) => {
  const { control, setValue } = useFormContext<CreateInvitationFormData>();
  const { errors } = useFormState({ control });
  const workspaceId = useWatch({ control, name: `grants.${index}.workspace_id` });
  const allow = useWatch({ control, name: `grants.${index}.allow` }) ?? [];
  const deny = useWatch({ control, name: `grants.${index}.deny` }) ?? [];
  const { data: workspaces } = useFetchWorkspaces();
  const { data: roles } = useFetchWorkspaceRoles(workspaceId);
  const { data: catalog } = useFetchPermissionCatalog();
  const assignableRoles = (roles ?? []).filter((role) => !role.is_owner_role);
  const roleId = useWatch({ control, name: `grants.${index}.role_id` });
  const grantErrors = errors.grants?.[index];

  useEffect(() => {
    if (!assignableRoles.some((role) => role.id === roleId) && assignableRoles[0]) setValue(`grants.${index}.role_id`, assignableRoles[0].id);
  }, [assignableRoles, index, roleId, setValue]);

  return (
    <li className="space-y-1">
      <div className="flex min-h-11 flex-wrap items-center gap-2 border-b border-border py-1.5">
        <Controller
          control={control}
          name={`grants.${index}.workspace_id`}
          render={({ field }) => (
            <Select value={field.value} onValueChange={field.onChange}>
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
        <Controller
          control={control}
          name={`grants.${index}.role_id`}
          render={({ field }) => (
            <Select value={field.value} onValueChange={field.onChange} disabled={assignableRoles.length === 0}>
              <SelectTrigger aria-label={`Role ${index + 1}`} className="h-8 w-auto min-w-28 shrink-0"><SelectValue placeholder="Choose a role" /></SelectTrigger>
              <SelectContent>{assignableRoles.map((role) => <SelectItem key={role.id} value={role.id}>{role.name}</SelectItem>)}</SelectContent>
            </Select>
          )}
        />
        <button type="button" aria-label={`Remove workspace ${index + 1}`} className="text-sm text-muted-foreground hover:text-foreground" onClick={() => remove(index)}>Remove</button>
      </div>
      {grantErrors?.workspace_id?.message && <p role="alert" className="text-sm text-destructive">{grantErrors.workspace_id.message}</p>}
      {grantErrors?.role_id?.message && <p role="alert" className="text-sm text-destructive">{grantErrors.role_id.message}</p>}
      {workspaceId && <InvitationProjectAccess index={index} />}
      <Collapsible>
        <CollapsibleTrigger type="button" className="text-sm text-muted-foreground underline-offset-4 hover:text-foreground hover:underline">Optional permission overrides</CollapsibleTrigger>
        <CollapsibleContent className="mt-3 space-y-3">
          <p className="text-xs text-muted-foreground">These workspace-wide overrides apply on top of the selected role.</p>
          {catalog && <div className="space-y-1"><p className="text-xs font-medium">Allow</p><PermissionLevels entries={catalog} value={allow} onChange={(value) => setValue(`grants.${index}.allow`, value)} /></div>}
          {catalog && <div className="space-y-1"><p className="text-xs font-medium">Deny</p><PermissionGrid entries={catalog} value={deny} onChange={(value) => setValue(`grants.${index}.deny`, value)} /></div>}
        </CollapsibleContent>
      </Collapsible>
      {grantErrors?.allow?.message && <p role="alert" className="text-sm text-destructive">{grantErrors.allow.message}</p>}
    </li>
  );
};
