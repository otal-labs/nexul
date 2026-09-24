import { zodResolver } from "@hookform/resolvers/zod";
import { useForm } from "react-hook-form";

import { AdvancedFields } from "@/components/dns/AdvancedFields";
import { FormInput } from "@/components/FormInput";
import { TunnelPrerequisiteAlert } from "@/components/pairing/TunnelPrerequisiteAlert";
import { Button } from "@/components/ui/button";
import { errorMessage } from "@/api/client";
import { tunnelPrerequisite, useCreateComputerTunnel } from "@/hooks/PairingHooks";
import {
  CreateComputerTunnelFormSchema,
  DEFAULT_T3_CODE_PORT,
  type Computer,
  type CreateComputerTunnelFormData,
} from "@/models/Pairing";

interface NameComputerFormProps {
  onCreated: (computer: Computer) => void;
}

// Naming the computer creates its tunnel; a missing instance prerequisite shows its fix right here.
export const NameComputerForm = ({ onCreated }: NameComputerFormProps) => {
  const create = useCreateComputerTunnel();
  const form = useForm<CreateComputerTunnelFormData>({
    resolver: zodResolver(CreateComputerTunnelFormSchema),
    defaultValues: { name: "", port: DEFAULT_T3_CODE_PORT },
  });
  const submit = form.handleSubmit(async (input) => {
    const computer = await create.mutateAsync(input).catch(() => undefined);
    if (computer) onCreated(computer);
  });
  const prerequisite = tunnelPrerequisite(create.error);

  return (
    <form onSubmit={submit} className="space-y-4">
      {prerequisite && <TunnelPrerequisiteAlert reason={prerequisite} onRetry={submit} retrying={create.isPending} />}
      <FormInput control={form.control} name="name" label="Computer name" placeholder="e.g. Work laptop" autoFocus />
      <AdvancedFields>
        <FormInput control={form.control} name="port" label="T3 Code port" type="number" inputMode="numeric" />
      </AdvancedFields>
      {create.error && !prerequisite && (
        <p role="alert" className="text-sm text-destructive">
          {errorMessage(create.error)}
        </p>
      )}
      <Button type="submit" disabled={create.isPending}>
        {create.isPending ? "Creating tunnel…" : "Create tunnel"}
      </Button>
    </form>
  );
};
