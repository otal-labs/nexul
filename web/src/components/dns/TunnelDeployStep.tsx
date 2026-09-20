import { zodResolver } from "@hookform/resolvers/zod";
import { useState } from "react";
import { useForm } from "react-hook-form";
import { z } from "zod";

import { AdvancedFields } from "@/components/dns/AdvancedFields";
import { TunnelDeployWatch } from "@/components/dns/TunnelDeployWatch";
import { ErrorDisplay } from "@/components/ErrorDisplay";
import { FormInput } from "@/components/FormInput";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { RunnerPicker } from "@/components/RunnerPicker";
import { FormSelect } from "@/components/ticket/FormSelect";
import { Button } from "@/components/ui/button";
import { useCreateTunnel, useProvisionTunnelAgent } from "@/hooks/DnsHooks";
import { useFetchProjects } from "@/hooks/ProjectHooks";
import { useRunners } from "@/hooks/RunnerHooks";
import { soleItem, type TunnelDeployment } from "@/models/DNS";
import type { Project } from "@/models/Project";
import type { Runner } from "@/models/Runner";

const TunnelDeploySchema = z.object({
  tunnel_name: z.string().trim().min(1, "Tunnel name is required"),
  project_id: z.string().min(1, "Choose a project"),
  target: z.string().trim().min(1, "Runner is required"),
  docker_network: z.string().trim().min(1, "Docker network is required"),
});

type TunnelDeployFormData = z.infer<typeof TunnelDeploySchema>;

interface TunnelDeployFieldsProps {
  projects: Project[];
  runners: Runner[];
  onConnected: (deployment: TunnelDeployment) => void;
}

// Mounted only once the lists are loaded so defaultValues can preselect the sole project and runner.
const TunnelDeployFields = ({ projects, runners, onConnected }: TunnelDeployFieldsProps) => {
  const createTunnel = useCreateTunnel();
  const provisionAgent = useProvisionTunnelAgent();
  const [pending, setPending] = useState<TunnelDeployment | null>(null);
  const form = useForm<TunnelDeployFormData>({
    defaultValues: {
      tunnel_name: "instance",
      project_id: soleItem(projects)?.id ?? "",
      target: soleItem(runners.filter((r) => r.connected))?.name ?? "",
      docker_network: "nexul_default",
    },
    resolver: zodResolver(TunnelDeploySchema),
  });
  const busy = createTunnel.isPending || provisionAgent.isPending;

  const onSubmit = async (data: TunnelDeployFormData) => {
    try {
      const tunnel = await createTunnel.mutateAsync({ name: data.tunnel_name });
      const agent = await provisionAgent.mutateAsync({
        tunnel_id: tunnel.id,
        project_id: data.project_id,
        target: data.target,
        docker_network: data.docker_network,
      });
      setPending({ tunnelId: tunnel.id, tunnelName: tunnel.name, serviceId: agent.service_id, target: data.target });
    } catch {
      // Errors surface through the hooks' toasts; both calls are retry-safe.
    }
  };

  return (
    <>
      {pending && <TunnelDeployWatch deployment={pending} onConnected={onConnected} onRetry={() => setPending(null)} />}
      {!pending && (
        <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-5">
          <FormInput control={form.control} name="tunnel_name" label="Tunnel name" placeholder="instance" />
          <p className="font-mono text-xs text-muted-foreground">
            cloudflared-{form.watch("tunnel_name") || "instance"} on {form.watch("target") || "runner"}
          </p>
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
          <Button type="submit" className="w-full sm:w-auto" disabled={busy}>
            {busy ? "Deploying tunnel…" : "Deploy tunnel"}
          </Button>
        </form>
      )}
    </>
  );
};

interface TunnelDeployStepProps {
  onConnected: (deployment: TunnelDeployment) => void;
}

export const TunnelDeployStep = ({ onConnected }: TunnelDeployStepProps) => {
  const { data: projects, isPending: projectsPending, error: projectsError } = useFetchProjects();
  const { data: runners, isPending: runnersPending, error: runnersError } = useRunners();

  return (
    <>
      {(projectsPending || runnersPending) && <LoadingDisplay />}
      {projectsError && <ErrorDisplay error={projectsError} />}
      {runnersError && <ErrorDisplay error={runnersError} />}
      {projects && runners && <TunnelDeployFields projects={projects} runners={runners} onConnected={onConnected} />}
    </>
  );
};
