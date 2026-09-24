import { zodResolver } from "@hookform/resolvers/zod";
import { useState } from "react";
import { useForm } from "react-hook-form";
import { z } from "zod";

import { api, errorMessage } from "@/api/client";
import { FormInput } from "@/components/FormInput";
import { TickerRow } from "@/components/TickerRow";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog";
import { useSaveManualCredentials, useVerifyManualCredentials } from "@/hooks/ConnectorsHooks";
import { useTicker } from "@/hooks/useTicker";
import type { Connector, CredentialCheck } from "@/models/Connectors";

interface ManualConnectorDialogProps {
  connector: Connector;
}

// A connector without named checks verifies as one row, shown only once Verify runs.
const wholeCheck: CredentialCheck = { key: "", label: "accepted these credentials" };

// Two beats: Verify fans one request per named check out in parallel and ticks each row as it lands; Confirm stores and closes.
export const ManualConnectorDialog = ({ connector }: ManualConnectorDialogProps) => {
  const fields = connector.manual ?? [];
  const named = !!connector.checks?.length;
  const checks = named ? connector.checks! : [wholeCheck];
  const verifyAll = useVerifyManualCredentials();
  const saveManual = useSaveManualCredentials();
  const [open, setOpen] = useState(false);

  const schema = z.object(
    Object.fromEntries(fields.map((f) => [f.key, z.string().trim().min(1, `${f.label} is required`)])),
  );
  type FormData = z.infer<typeof schema>;

  const form = useForm<FormData>({
    defaultValues: Object.fromEntries(fields.map((f) => [f.key, ""])) as FormData,
    resolver: zodResolver(schema),
  });

  const { fresh, verified, verifying, verify, reset, outcomeFor } = useTicker(
    checks,
    form.watch(),
    (key, data) =>
      key === ""
        ? verifyAll.mutateAsync({ id: connector.id, fields: data })
        : api.post(`/api/connectors/${connector.id}/manual/verify`, data, { params: { check: key } }),
  );

  const onConfirm = async (data: FormData) => {
    try {
      await saveManual.mutateAsync({ id: connector.id, fields: data });
      setOpen(false);
      form.reset();
      reset();
    } catch (err) {
      form.setError("root", { message: errorMessage(err) });
    }
  };

  return (
    <Dialog
      open={open}
      onOpenChange={(next) => {
        setOpen(next);
        if (!next) {
          form.reset();
          reset();
        }
      }}
    >
      <DialogTrigger asChild>
        <Button size="sm">Connect</Button>
      </DialogTrigger>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Connect {connector.name}</DialogTitle>
          <DialogDescription>
            {connector.description}
            {connector.docs_url && (
              <>
                {" "}
                <a href={connector.docs_url} target="_blank" rel="noreferrer" className="underline underline-offset-4">
                  Where to get this
                </a>
              </>
            )}
          </DialogDescription>
        </DialogHeader>
        <form onSubmit={form.handleSubmit(verified ? onConfirm : verify)} className="space-y-4">
          {fields.map((f) => (
            <div key={f.key} className="space-y-1.5">
              <FormInput
                control={form.control}
                name={f.key}
                id={`${connector.id}-${f.key}`}
                label={f.label}
                type={f.secret ? "password" : "text"}
                autoComplete="off"
                data-bwignore={f.secret ? "true" : undefined}
                data-1p-ignore={f.secret ? "true" : undefined}
                data-lpignore={f.secret ? "true" : undefined}
                className="text-base"
              />
              {f.hint && <p className="text-xs text-muted-foreground">{f.hint}</p>}
            </div>
          ))}
          {(named || fresh) && (
            <ul className="space-y-2" aria-label={named ? "Permissions" : "Verification"}>
              {checks.map((c) => (
                <TickerRow
                  key={c.key}
                  label={c.key === "" ? `${connector.name} ${c.label}` : c.label}
                  why={c.why}
                  outcome={outcomeFor(c.key)}
                />
              ))}
            </ul>
          )}
          {form.formState.errors.root && (
            <p role="alert" className="text-sm text-destructive">
              {form.formState.errors.root.message}
            </p>
          )}
          <DialogFooter>
            <Button type="submit" disabled={form.formState.isSubmitting || verifying}>
              {form.formState.isSubmitting && verified && "Connecting…"}
              {verifying && "Verifying…"}
              {!form.formState.isSubmitting && !verifying && (verified ? "Confirm" : "Verify")}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
};
