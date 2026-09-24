import { useFormDialogContext } from "@/components/dialogs/FormDialogContext";
import { FormInput } from "@/components/FormInput";
import { useRepairComputer } from "@/hooks/PairingHooks";
import type { Computer, PairComputerFormData } from "@/models/Pairing";

interface PairComputerFormProps {
  computer: Computer;
}

// Re-pairs a computer in place; a computer tunnel keeps its hostname, so only the URL of a computer paired by URL can change.
export const PairComputerForm = ({ computer }: PairComputerFormProps) => {
  const { control, onSubmit } = useFormDialogContext<PairComputerFormData>();
  const repair = useRepairComputer(computer);

  onSubmit(async (input) => {
    const repaired = await repair.mutateAsync(input);
    return { name: repaired.name, server_url: repaired.server_url, token: "" };
  });

  return (
    <div className="space-y-4">
      <FormInput control={control} name="name" label="Name" placeholder="e.g. Home, VPS" autoFocus />
      <FormInput
        control={control}
        name="server_url"
        label="T3 server URL"
        placeholder="https://your-t3-host:port"
        readOnly={!!computer.tunnel}
        className="font-mono text-xs"
      />
      <FormInput
        control={control}
        name="token"
        label="One-time pairing token"
        placeholder="Paste the token printed by `t3 pair`"
        autoComplete="off"
      />
    </div>
  );
};
