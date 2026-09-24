import { zodResolver } from "@hookform/resolvers/zod";
import { useForm } from "react-hook-form";

import { FormInput } from "@/components/FormInput";
import { Button } from "@/components/ui/button";
import { errorMessage } from "@/api/client";
import { pairFieldErrors, usePairComputer } from "@/hooks/PairingHooks";
import { PairComputerFormSchema, type Computer, type PairComputerFormData } from "@/models/Pairing";
import { cn } from "@/lib/utils";

interface PairT3CodeFormProps {
  // The computer tunnel from the Connect step; absent when pairing a machine by URL.
  computer?: Computer | undefined;
  onPaired: (computer: Computer) => void;
}

// Over a tunnel only the token is typed: the name and verified hostname are the Connect step's, read-only here.
export const PairT3CodeForm = ({ computer, onPaired }: PairT3CodeFormProps) => {
  const pair = usePairComputer();
  const form = useForm<PairComputerFormData>({
    resolver: zodResolver(PairComputerFormSchema),
    defaultValues: { name: computer?.name ?? "", server_url: computer?.server_url ?? "", token: "" },
  });
  const submit = form.handleSubmit(async (input) => {
    const paired = await pair.mutateAsync({ computerId: computer?.id, form: input }).catch((error: unknown) => {
      const fields = pairFieldErrors(error);
      fields.forEach(([field, message]) => form.setError(field, { type: "server", message }, { shouldFocus: true }));
      if (fields.length === 0) form.setError("root", { type: "server", message: errorMessage(error) });
      return undefined;
    });
    if (paired) onPaired(paired);
  });

  return (
    <form onSubmit={submit} className="space-y-4">
      <FormInput
        control={form.control}
        name="name"
        label="Name"
        placeholder="e.g. Home, VPS"
        readOnly={!!computer}
        autoFocus={!computer}
        className={cn(computer && "text-muted-foreground")}
      />
      <FormInput
        control={form.control}
        name="server_url"
        label="T3 server URL"
        placeholder="https://your-t3-host:port"
        readOnly={!!computer}
        className={cn("font-mono text-xs", computer && "text-muted-foreground")}
      />
      <FormInput
        control={form.control}
        name="token"
        label="One-time pairing token"
        placeholder="Paste the token printed by `t3 pair`"
        autoComplete="off"
        autoFocus={!!computer}
      />
      {form.formState.errors.root && (
        <p role="alert" className="text-sm text-destructive">
          {form.formState.errors.root.message}
        </p>
      )}
      <Button type="submit" disabled={pair.isPending}>
        {pair.isPending ? "Pairing…" : "Pair T3 Code"}
      </Button>
    </form>
  );
};
