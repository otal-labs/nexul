import { useFormDialogContext } from "@/components/dialogs/FormDialogContext";
import { FormInput } from "@/components/FormInput";
import { usePairComputer, useRepairComputer } from "@/hooks/PairingHooks";
import type { PairComputerFormData } from "@/models/Pairing";

interface PairComputerFormProps {
  // Set only for re-pair: routes submit through useRepairComputer(id) instead of usePairComputer.
  computerId?: string;
}

export const PairComputerForm = ({ computerId }: PairComputerFormProps) => {
  const { control, onSubmit } = useFormDialogContext<PairComputerFormData>();
  const pair = usePairComputer();
  const repair = useRepairComputer(computerId ?? "");

  onSubmit(async (input) => {
    const computer = computerId ? await repair.mutateAsync(input) : await pair.mutateAsync(input);
    return { name: computer.name, server_url: computer.server_url, token: "" };
  });

  return (
    <div className="space-y-4">
      <FormInput control={control} name="name" label="Name" placeholder="e.g. Home, VPS" autoFocus />
      <FormInput
        control={control}
        name="server_url"
        label="T3 server URL"
        placeholder="https://your-t3-host:port"
      />
      <FormInput
        control={control}
        name="token"
        label="One-time pairing token"
        placeholder="Paste the token printed by `t3 pair`"
      />
    </div>
  );
};
