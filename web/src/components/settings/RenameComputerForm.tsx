import { useFormDialogContext } from "@/components/dialogs/FormDialogContext";
import { FormInput } from "@/components/FormInput";
import { useRenameComputer } from "@/hooks/ComputerHooks";
import { pairFieldErrors } from "@/hooks/PairingHooks";
import type { RenameComputerFormData } from "@/models/Pairing";

interface RenameComputerFormProps {
  computerId: string;
}

// A computer's name, which starts as the hostname its app reported.
export const RenameComputerForm = ({ computerId }: RenameComputerFormProps) => {
  const { control, setError, onSubmit } = useFormDialogContext<RenameComputerFormData>();
  const rename = useRenameComputer(computerId);

  onSubmit(async (input) => {
    const renamed = await rename.mutateAsync(input).catch((error: unknown) => {
      const message = pairFieldErrors(error).find(([field]) => field === "name")?.[1];
      if (message) setError("name", { type: "server", message });
      throw error;
    });
    return { name: renamed.name };
  });

  return <FormInput control={control} name="name" label="Name" placeholder="e.g. Work laptop" autoFocus />;
};
