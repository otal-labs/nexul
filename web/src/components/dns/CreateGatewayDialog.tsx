import { zodResolver } from "@hookform/resolvers/zod";
import { useState } from "react";
import { useForm, type UseFormReturn } from "react-hook-form";

import { ErrorDisplay } from "@/components/ErrorDisplay";
import { FormInput } from "@/components/FormInput";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { RunnerPicker } from "@/components/RunnerPicker";
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
import { FormSelect } from "@/components/ticket/FormSelect";
import { useCreateGateway, useFetchDnsZones, useFetchTunnels } from "@/hooks/DnsHooks";
import { useFetchProjects } from "@/hooks/ProjectHooks";
import { CreateGatewayFormSchema, GatewayKinds, type CreateGatewayFormData, type Tunnel, type Zone } from "@/models/DNS";
import type { Project } from "@/models/Project";

const kindLabel = (kind: (typeof GatewayKinds)[number]) => (kind === "tunnel" ? "Cloudflare tunnel" : "Reverse proxy");
const KIND_OPTIONS = GatewayKinds.map((kind) => ({ value: kind, label: kindLabel(kind) }));

interface CreateGatewayFormFieldsProps {
  form: UseFormReturn<CreateGatewayFormData>;
  kind: CreateGatewayFormData["kind"];
  zones: Zone[];
  tunnels: Tunnel[];
  projects: Project[];
}

// Kind picks what's provisioned: cloudflared (tunnel, references a dns tunnel) or Traefik (proxy, an address).
const CreateGatewayFormFields = ({ form, kind, zones, tunnels, projects }: CreateGatewayFormFieldsProps) => {
  const zoneName = (id: string) => zones.find((z) => z.id === id)?.name ?? "";
  return (
    <>
      <FormSelect control={form.control} name="kind" label="Kind" options={KIND_OPTIONS} />
      <FormInput control={form.control} name="docker_network" label="Docker network" placeholder="nexul" className="text-base" />
      {kind === "tunnel" && (
        <FormSelect
          control={form.control}
          name="tunnel_id"
          label="Tunnel"
          placeholder="Choose a tunnel…"
          options={tunnels.map((t) => ({ value: t.id, label: t.name }))}
        />
      )}
      {kind === "proxy" && (
        <FormInput
          control={form.control}
          name="server_address"
          label="Server address (the proxy listens here)"
          placeholder="203.0.113.10"
          className="text-base"
        />
      )}
      <FormSelect
        control={form.control}
        name="zone_id"
        label="Zone"
        placeholder="Choose a zone…"
        options={zones.map((z) => ({ value: z.id, label: z.name }))}
        onChangeValue={(value) => form.setValue("zone", zoneName(value), { shouldValidate: true })}
      />
      <FormSelect
        control={form.control}
        name="project_id"
        label="Project"
        placeholder="Choose a project…"
        options={projects.map((p) => ({ value: p.id, label: p.name }))}
      />
      <RunnerPicker control={form.control} name="target" />
    </>
  );
};

export const CreateGatewayDialog = () => {
  const [open, setOpen] = useState(false);
  const { data: projects, isPending: projectsPending, error: projectsError } = useFetchProjects();
  const { data: zones, isPending: zonesPending, error: zonesError } = useFetchDnsZones(open);
  const { data: tunnels, isPending: tunnelsPending, error: tunnelsError } = useFetchTunnels(open);
  const createGateway = useCreateGateway();

  const form = useForm<CreateGatewayFormData>({
    defaultValues: {
      kind: "tunnel",
      docker_network: "",
      zone_id: "",
      zone: "",
      tunnel_id: "",
      server_address: "",
      project_id: "",
      target: "",
    },
    resolver: zodResolver(CreateGatewayFormSchema),
  });

  const kind = form.watch("kind");
  const loading = projectsPending || zonesPending || tunnelsPending;
  const loadError = projectsError ?? zonesError ?? tunnelsError;

  const onSubmit = async (data: CreateGatewayFormData) => {
    await createGateway.mutateAsync(data);
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
        <Button size="sm">Create gateway</Button>
      </DialogTrigger>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Create a gateway</DialogTitle>
          <DialogDescription>
            Give a docker network internet reachability via a Cloudflare tunnel or a reverse proxy.
          </DialogDescription>
        </DialogHeader>
        {loading && <LoadingDisplay />}
        {loadError && <ErrorDisplay error={loadError} />}
        {projects && zones && tunnels && (
          <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-4">
            <CreateGatewayFormFields form={form} kind={kind} zones={zones} tunnels={tunnels} projects={projects} />
            <DialogFooter>
              <Button type="submit" disabled={createGateway.isPending}>
                {createGateway.isPending ? "Creating…" : "Create gateway"}
              </Button>
            </DialogFooter>
          </form>
        )}
      </DialogContent>
    </Dialog>
  );
};
