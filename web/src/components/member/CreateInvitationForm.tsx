import { useEffect } from "react";
import { useFieldArray, useFormContext, useFormState, useWatch, Controller } from "react-hook-form";
import { PlusIcon } from "lucide-react";

import { Microheader } from "@/components/Microheader";
import { Button } from "@/components/ui/button";
import { useFormDialogContext } from "@/components/dialogs/FormDialogContext";
import { InvitationGrantRow } from "@/components/member/InvitationGrantRow";
import { RadioGroup, RadioGroupItem } from "@/components/ui/radio-group";
import { useCreateInvitation } from "@/hooks/InvitationHooks";
import { hasDuplicateInvitationWorkspaces, type CreateInvitationFormData, type CreatedInvitation } from "@/models/Invitation";

interface CreateInvitationFormProps {
  onCreated: (invitation: CreatedInvitation) => void;
}

export const CreateInvitationForm = ({ onCreated }: CreateInvitationFormProps) => {
  const { control } = useFormContext<CreateInvitationFormData>();
  const { errors } = useFormState({ control });
  const { fields, append, remove } = useFieldArray({ control, name: "grants" });
  const { onSubmit, setError, setBlocked } = useFormDialogContext<CreateInvitationFormData>();
  const create = useCreateInvitation();
  const grants = useWatch({ control, name: "grants" });
  const roleless = grants.some((grant) => grant.role_ids.length === 0);

  // Create link waits until every workspace gives a role; the shell owns the button, so it is told.
  useEffect(() => {
    setBlocked(roleless);
  }, [roleless, setBlocked]);

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
    <div className="scroll-edge max-h-[min(60vh,32rem)] space-y-5 overflow-x-hidden overflow-y-auto pr-1 pb-6">
      {errors.grants?.message && <p role="alert" className="text-sm text-destructive">{errors.grants.message}</p>}
      {errors.root?.serverError?.message && <p role="alert" className="text-sm text-destructive">{errors.root.serverError.message}</p>}
      <fieldset className="space-y-2"><legend className="text-sm font-medium">Link expires in</legend><Controller control={control} name="expires_in_days" render={({ field }) => <RadioGroup value={String(field.value)} onValueChange={(value) => field.onChange(Number(value))} className="grid grid-cols-2 gap-2"><label className="flex items-center gap-2 rounded-md border p-3 text-sm"><RadioGroupItem value="1" />1 day</label><label className="flex items-center gap-2 rounded-md border p-3 text-sm"><RadioGroupItem value="7" />7 days</label></RadioGroup>} /></fieldset>
      <div className="space-y-2"><div className="flex items-center justify-between"><Microheader>Workspace access</Microheader><Button type="button" variant="ghost" size="sm" className="-mr-2" onClick={() => append({ workspace_id: "", role_ids: [] })}><PlusIcon aria-hidden />Add workspace</Button></div><ul className="space-y-5">{fields.map((field, index) => <InvitationGrantRow key={field.id} index={index} remove={remove} removable={fields.length > 1} />)}</ul></div>
    </div>
  );
};
