import { useFormDialogContext } from "@/components/dialogs/FormDialogContext";
import { FormInput } from "@/components/FormInput";
import { useRenamePasskey } from "@/hooks/PasskeyHooks";
import type { Passkey, RenamePasskeyFormData } from "@/models/Passkey";

interface RenamePasskeyFormProps {
  passkey: Passkey;
}

export const RenamePasskeyForm = ({ passkey }: RenamePasskeyFormProps) => {
  const { control, onSubmit } = useFormDialogContext<RenamePasskeyFormData>();
  const rename = useRenamePasskey();

  onSubmit(async (input) => {
    await rename.mutateAsync({ id: passkey.id, name: input.name });
    return input;
  });

  return <FormInput control={control} name="name" id="passkey-name" label="Name" autoFocus autoComplete="off" />;
};
