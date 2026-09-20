import { zodResolver } from "@hookform/resolvers/zod";
import { useState } from "react";
import { useForm } from "react-hook-form";

import { ErrorDisplay } from "@/components/ErrorDisplay";
import { FormInput } from "@/components/FormInput";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { FormSelect } from "@/components/ticket/FormSelect";
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
import { useCreateExposure, useFetchDnsZones } from "@/hooks/DnsHooks";
import { ExposeServiceFormSchema, type ExposeServiceFormData, type Zone } from "@/models/DNS";
import type { Container } from "@/models/Stack";

interface ExposeServiceDialogProps {
  containers: Container[];
}

// A container's first published port, parsed from its "host:container" or bare "container" port strings.
const firstPort = (c: Container): number | undefined => {
  const raw = c.ports?.[0]?.split(":").pop();
  const port = raw ? Number(raw) : NaN;
  return Number.isFinite(port) && port > 0 ? port : undefined;
};

// The gateway is resolved (reused or provisioned) by the backend now (spec §7) — this only picks the container,
// port, hostname, and zone.
export const ExposeServiceDialog = ({ containers }: ExposeServiceDialogProps) => {
  const [open, setOpen] = useState(false);
  const { data: zones, isPending: zonesPending, error: zonesError } = useFetchDnsZones(open);
  const createExposure = useCreateExposure();

  const defaultContainer = containers.find((c) => firstPort(c) != null) ?? containers[0];

  const form = useForm<ExposeServiceFormData>({
    defaultValues: {
      hostname: "",
      service_id: defaultContainer?.id ?? "",
      port: (defaultContainer ? firstPort(defaultContainer) : undefined) ?? (undefined as unknown as number),
      zone_id: "",
      zone: "",
    },
    resolver: zodResolver(ExposeServiceFormSchema),
  });

  const zoneName = (id: string) => zones?.find((z: Zone) => z.id === id)?.name ?? "";

  const onSubmit = async (data: ExposeServiceFormData) => {
    await createExposure.mutateAsync(data);
    setOpen(false);
    form.reset();
  };

  return (
    <Dialog
      open={open}
      onOpenChange={(next) => {
        setOpen(next);
        if (!next) form.reset();
      }}
    >
      <DialogTrigger asChild>
        <Button size="sm">Expose</Button>
      </DialogTrigger>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Expose a container</DialogTitle>
          <DialogDescription>Routes a hostname to one of this stack's containers.</DialogDescription>
        </DialogHeader>
        {zonesPending && <LoadingDisplay />}
        {zonesError && <ErrorDisplay error={zonesError} />}
        {zones && (
          <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-4">
            <FormSelect
              control={form.control}
              name="service_id"
              label="Container"
              options={containers.map((c) => ({ value: c.id, label: c.container_name || c.name }))}
            />
            <div>
              <FormInput control={form.control} name="hostname" label="Hostname" placeholder="app.example.com" className="text-base" />
            </div>
            <FormInput control={form.control} name="port" label="Container port" type="number" placeholder="8080" className="text-base" />
            <FormSelect
              control={form.control}
              name="zone_id"
              label="Zone"
              placeholder="Choose a zone…"
              options={zones.map((z) => ({ value: z.id, label: z.name }))}
              onChangeValue={(value) => form.setValue("zone", zoneName(value), { shouldValidate: true })}
            />
            <DialogFooter>
              <Button type="submit" disabled={createExposure.isPending}>
                {createExposure.isPending ? "Exposing…" : "Expose"}
              </Button>
            </DialogFooter>
          </form>
        )}
      </DialogContent>
    </Dialog>
  );
};
