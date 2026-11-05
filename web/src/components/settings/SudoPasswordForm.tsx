import { useFormDialogContext } from "@/components/dialogs/FormDialogContext";
import { FormInput } from "@/components/FormInput";
import type { SudoPasswordFormData } from "@/models/ComputerTools";

interface SudoPasswordFormProps {
  install: (sudoPassword: string) => Promise<void>;
}

// The sudo password an install needs on a computer; OK runs the install, and a wrong password keeps the dialog open.
export const SudoPasswordForm = ({ install }: SudoPasswordFormProps) => {
  const { control, onSubmit } = useFormDialogContext<SudoPasswordFormData>();

  onSubmit(async ({ password }) => {
    await install(password);
    return { password: "" };
  });

  return <FormInput control={control} name="password" label="Sudo password" type="password" autoComplete="off" data-1p-ignore autoFocus />;
};
