import { useEffect } from "react";
import { useFieldArray, useFormContext, useFormState, useWatch, Controller } from "react-hook-form";

import { PermissionGrid } from "@/components/access/PermissionGrid";
import { useFormDialogContext } from "@/components/dialogs/FormDialogContext";
import { Collapsible, CollapsibleContent, CollapsibleTrigger } from "@/components/ui/collapsible";
import { RadioGroup, RadioGroupItem } from "@/components/ui/radio-group";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { useFetchPermissionCatalog } from "@/hooks/PermissionHooks";
import { useFetchWorkspaceRoles } from "@/hooks/RoleHooks";
import { useFetchWorkspaces } from "@/hooks/WorkspaceHooks";
import { useCreateInvitation } from "@/hooks/InvitationHooks";
import { hasDuplicateInvitationWorkspaces, type CreateInvitationFormData, type CreatedInvitation } from "@/models/Invitation";

interface CreateInvitationFormProps {
  onCreated: (invitation: CreatedInvitation) => void;
}

const InvitationGrantRow = ({ index, remove }: { index: number; remove: (index: number) => void }) => {
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
    <li className="space-y-3 rounded-md border bg-card p-3">
      <div className="flex items-start gap-2">
        <div className="min-w-0 flex-1 space-y-2">
          <label htmlFor={`invite-workspace-${index}`} className="text-sm font-medium">Workspace</label>
          <Controller
            control={control}
            name={`grants.${index}.workspace_id`}
            render={({ field }) => (
              <Select value={field.value} onValueChange={field.onChange}>
                <SelectTrigger id={`invite-workspace-${index}`} aria-label={`Workspace ${index + 1}`}>
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
          {grantErrors?.workspace_id?.message && <p role="alert" className="text-sm text-destructive">{grantErrors.workspace_id.message}</p>}
        </div>
        <button type="button" aria-label={`Remove workspace ${index + 1}`} className="mt-8 text-sm text-muted-foreground hover:text-foreground" onClick={() => remove(index)}>Remove</button>
      </div>
      <div className="space-y-2">
        <label htmlFor={`invite-role-${index}`} className="text-sm font-medium">Role</label>
        <Controller
          control={control}
          name={`grants.${index}.role_id`}
          render={({ field }) => (
            <Select value={field.value} onValueChange={field.onChange} disabled={assignableRoles.length === 0}>
              <SelectTrigger id={`invite-role-${index}`} aria-label={`Role ${index + 1}`}><SelectValue placeholder="Choose a role" /></SelectTrigger>
              <SelectContent>{assignableRoles.map((role) => <SelectItem key={role.id} value={role.id}>{role.name}</SelectItem>)}</SelectContent>
            </Select>
          )}
          />
        {grantErrors?.role_id?.message && <p role="alert" className="text-sm text-destructive">{grantErrors.role_id.message}</p>}
      </div>
      <Collapsible>
        <CollapsibleTrigger type="button" className="text-sm text-muted-foreground underline-offset-4 hover:text-foreground hover:underline">Optional permission overrides</CollapsibleTrigger>
        <CollapsibleContent className="mt-3 space-y-3">
          <p className="text-xs text-muted-foreground">These workspace-wide overrides apply on top of the selected role.</p>
          {catalog && (
            <>
              <div className="space-y-1"><p className="text-xs font-medium">Allow</p><PermissionGrid entries={catalog} value={allow} onChange={(value) => setValue(`grants.${index}.allow`, value)} /></div>
              <div className="space-y-1"><p className="text-xs font-medium">Deny</p><PermissionGrid entries={catalog} value={deny} onChange={(value) => setValue(`grants.${index}.deny`, value)} /></div>
            </>
          )}
        </CollapsibleContent>
      </Collapsible>
      {grantErrors?.allow?.message && <p role="alert" className="text-sm text-destructive">{grantErrors.allow.message}</p>}
    </li>
  );
};

export const CreateInvitationForm = ({ onCreated }: CreateInvitationFormProps) => {
  const { control } = useFormContext<CreateInvitationFormData>();
  const { errors } = useFormState({ control });
  const { fields, append, remove } = useFieldArray({ control, name: "grants" });
  const { onSubmit, setError } = useFormDialogContext<CreateInvitationFormData>();
  const create = useCreateInvitation();

  onSubmit(async (input) => {
    if (hasDuplicateInvitationWorkspaces(input.grants)) {
      setError("root.serverError", { type: "validate", message: "Choose each workspace only once" });
      throw new Error("Choose each workspace only once");
    }
    const invitation = await create.mutateAsync(input);
    onCreated(invitation);
    return input;
  });

  return (
    <div className="max-h-[min(60vh,32rem)] space-y-5 overflow-y-auto pr-1">
      {errors.grants?.message && <p role="alert" className="text-sm text-destructive">{errors.grants.message}</p>}
      {errors.root?.serverError?.message && <p role="alert" className="text-sm text-destructive">{errors.root.serverError.message}</p>}
      <fieldset className="space-y-2"><legend className="text-sm font-medium">Link expires in</legend><Controller control={control} name="expires_in_days" render={({ field }) => <RadioGroup value={String(field.value)} onValueChange={(value) => field.onChange(Number(value))} className="grid grid-cols-2 gap-2"><label className="flex items-center gap-2 rounded-md border p-3 text-sm"><RadioGroupItem value="1" />1 day</label><label className="flex items-center gap-2 rounded-md border p-3 text-sm"><RadioGroupItem value="7" />7 days</label></RadioGroup>} /></fieldset>
      <div className="space-y-2"><div className="flex items-center justify-between"><h3 className="text-sm font-medium">Workspace access</h3><button type="button" className="text-sm text-muted-foreground underline-offset-4 hover:text-foreground hover:underline" onClick={() => append({ workspace_id: "", role_id: "", allow: [], deny: [] })}>Add workspace</button></div><ul className="space-y-2">{fields.map((field, index) => <InvitationGrantRow key={field.id} index={index} remove={remove} />)}</ul></div>
    </div>
  );
};
