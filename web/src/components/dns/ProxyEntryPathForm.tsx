import { zodResolver } from "@hookform/resolvers/zod";
import { useForm } from "react-hook-form";
import { z } from "zod";

import { AdvancedFields } from "@/components/dns/AdvancedFields";
import { ErrorDisplay } from "@/components/ErrorDisplay";
import { FormInput } from "@/components/FormInput";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { RunnerPicker } from "@/components/RunnerPicker";
import { InstanceHostNotice } from "@/components/dns/InstanceHostNotice";
import { FormSelect } from "@/components/ticket/FormSelect";
import { Button } from "@/components/ui/button";
import { useFetchSettings } from "@/hooks/AuthHooks";
import { useCreateInstanceRecord, useFetchDnsZones, useProvisionReverseProxy } from "@/hooks/DnsHooks";
import { useFetchProjects } from "@/hooks/ProjectHooks";
import { useRunners } from "@/hooks/RunnerHooks";
import { instanceRecordName, soleItem, type DnsSetupResult, type Zone } from "@/models/DNS";
import type { Project } from "@/models/Project";
import type { Runner } from "@/models/Runner";

const ProxyEntryPathSchema = z.object({
  zone_id: z.string().min(1, "Choose a zone"),
  zone: z.string().min(1, "Choose a zone"),
  record_type: z.enum(["A", "AAAA"]),
  server_address: z.string().trim().min(1, "Server address is required"),
  project_id: z.string().min(1, "Choose a project"),
  target: z.string().trim().min(1, "Runner is required"),
  docker_network: z.string().trim().min(1, "Docker network is required"),
});

type ProxyEntryPathFormData = z.infer<typeof ProxyEntryPathSchema>;

interface ProxyEntryPathFieldsProps {
  projects: Project[];
  zones: Zone[];
  runners: Runner[];
  onDone: (result: DnsSetupResult) => void;
}

// Mounted only once the lists are loaded so defaultValues can preselect the sole zone, project, and runner.
const ProxyEntryPathFields = ({ projects, zones, runners, onDone }: ProxyEntryPathFieldsProps) => {
  const provision = useProvisionReverseProxy();
  const createInstanceRecord = useCreateInstanceRecord();
  const { data: settings } = useFetchSettings();
  const soleZone = soleItem(zones);
  const form = useForm<ProxyEntryPathFormData>({
    defaultValues: {
      zone_id: soleZone?.id ?? "",
      zone: soleZone?.name ?? "",
      record_type: "A",
      server_address: "",
      project_id: soleItem(projects)?.id ?? "",
      target: soleItem(runners.filter((r) => r.connected))?.name ?? "",
      docker_network: "nexul_default",
    },
    resolver: zodResolver(ProxyEntryPathSchema),
  });

  const zoneName = (id: string) => zones.find((z) => z.id === id)?.name ?? "";
  const zone = form.watch("zone");
  const serverAddress = form.watch("server_address");
  const busy = provision.isPending || createInstanceRecord.isPending;
  // Same rule as the bare path: the instance record's name comes from the saved instance URL.
  const blocked = !!zone && !!settings?.instance_url && instanceRecordName(settings.instance_url, zone) === null;

  const onSubmit = async (data: ProxyEntryPathFormData) => {
    try {
      await provision.mutateAsync({
        project_id: data.project_id,
        target: data.target,
        docker_network: data.docker_network,
      });
      await createInstanceRecord.mutateAsync({
        zone_id: data.zone_id,
        zone: data.zone,
        type: data.record_type,
        target: data.server_address,
      });
      onDone({
        headline: `Traefik runs on ${data.target} and the instance record under ${data.zone} points at it.`,
        detail: `${data.record_type} → ${data.server_address}`,
      });
    } catch {
      // Errors surface through the hooks' toasts; every call is retry-safe.
    }
  };

  return (
    <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-5">
      <div className="grid gap-4 sm:grid-cols-2">
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
          name="record_type"
          label="Record type"
          options={[
            { value: "A", label: "A" },
            { value: "AAAA", label: "AAAA" },
          ]}
        />
      </div>
      <FormInput
        control={form.control}
        name="server_address"
        label="Server address (the proxy listens here)"
        placeholder="203.0.113.10"
      />
      <InstanceHostNotice zone={zone} target={serverAddress} />
      <AdvancedFields>
        <FormSelect
          control={form.control}
          name="project_id"
          label="Project"
          placeholder="Choose a project…"
          options={projects.map((p) => ({ value: p.id, label: p.name }))}
        />
        <RunnerPicker control={form.control} name="target" />
        <FormInput control={form.control} name="docker_network" label="Docker network" placeholder="nexul_default" />
      </AdvancedFields>
      <Button type="submit" className="w-full sm:w-auto" disabled={busy || blocked}>
        {busy ? "Setting up proxy…" : "Set up reverse proxy"}
      </Button>
    </form>
  );
};

interface ProxyEntryPathFormProps {
  onDone: (result: DnsSetupResult) => void;
}

export const ProxyEntryPathForm = ({ onDone }: ProxyEntryPathFormProps) => {
  const { data: projects, isPending: projectsPending, error: projectsError } = useFetchProjects();
  const { data: zones, isPending: zonesPending, error: zonesError } = useFetchDnsZones(true);
  const { data: runners, isPending: runnersPending, error: runnersError } = useRunners();

  return (
    <>
      {(projectsPending || zonesPending || runnersPending) && <LoadingDisplay />}
      {projectsError && <ErrorDisplay error={projectsError} />}
      {zonesError && <ErrorDisplay error={zonesError} />}
      {runnersError && <ErrorDisplay error={runnersError} />}
      {projects && zones && runners && (
        <ProxyEntryPathFields projects={projects} zones={zones} runners={runners} onDone={onDone} />
      )}
    </>
  );
};
