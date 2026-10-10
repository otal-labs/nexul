import { zodResolver } from "@hookform/resolvers/zod";
import { LinkIcon } from "lucide-react";
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
  onPairByUrl: () => void;
}

// Naming the computer creates its tunnel; a missing instance prerequisite shows its fix right here.
export const NameComputerForm = ({ onCreated, onPairByUrl }: NameComputerFormProps) => {
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
    <form onSubmit={submit} className="space-y-5">
      {prerequisite && <TunnelPrerequisiteAlert reason={prerequisite} onRetry={submit} retrying={create.isPending} />}
      {!prerequisite && (
        <div className="space-y-4">
          <div className="space-y-1.5">
            <FormInput control={form.control} name="name" label="Computer name" placeholder="e.g. Work laptop" autoFocus />
            <p className="text-xs text-muted-foreground">Its tunnel gets a hostname made from this name.</p>
          </div>
          <AdvancedFields>
            <FormInput control={form.control} name="port" label="T3 Code port" type="number" inputMode="numeric" />
          </AdvancedFields>
          {create.error && (
            <p role="alert" className="text-sm text-destructive">
              {errorMessage(create.error)}
            </p>
          )}
          <Button type="submit" loading={create.isPending}>
            {create.isPending ? "Creating tunnel…" : "Create tunnel"}
          </Button>
        </div>
      )}
      <p className="flex flex-wrap items-center gap-x-1 border-t border-border pt-4 text-sm text-muted-foreground">
        Can this server already reach the machine, like a VPS?
        <Button type="button" variant="link" className="h-auto p-0 text-foreground" onClick={onPairByUrl}>
          <LinkIcon className="size-3.5" aria-hidden />
          Pair by URL
        </Button>
      </p>
    </form>
  );
};
