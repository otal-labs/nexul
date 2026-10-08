import { useFieldArray, useFormContext, useFormState, Controller } from "react-hook-form";

import { Microheader } from "@/components/access/Microheader";
import { useFormDialogContext } from "@/components/dialogs/FormDialogContext";
import { InvitationGrantRow } from "@/components/member/InvitationGrantRow";
import { RadioGroup, RadioGroupItem } from "@/components/ui/radio-group";
import { useCreateInvitation } from "@/hooks/InvitationHooks";
import { EveryProject } from "@/models/Team";
import { hasDuplicateInvitationWorkspaces, type CreateInvitationFormData, type CreatedInvitation } from "@/models/Invitation";

interface CreateInvitationFormProps {
  onCreated: (invitation: CreatedInvitation) => void;
}

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
      <div className="space-y-2"><div className="flex items-center justify-between"><Microheader>Workspace access</Microheader><button type="button" className="text-sm text-muted-foreground underline-offset-4 hover:text-foreground hover:underline" onClick={() => append({ workspace_id: "", role_id: "", allow: [], deny: [], every_project: EveryProject.Role, project_access: [] })}>Add workspace</button></div><ul className="space-y-5">{fields.map((field, index) => <InvitationGrantRow key={field.id} index={index} remove={remove} />)}</ul></div>
    </div>
  );
};
