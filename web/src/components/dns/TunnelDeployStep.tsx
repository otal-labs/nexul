import { zodResolver } from "@hookform/resolvers/zod";
import { useState } from "react";
import { useForm } from "react-hook-form";
import { z } from "zod";

import { AdvancedFields } from "@/components/dns/AdvancedFields";
import { TunnelDeployWatch } from "@/components/dns/TunnelDeployWatch";
import { ErrorDisplay } from "@/components/ErrorDisplay";
import { FormInput } from "@/components/FormInput";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { MachinePicker } from "@/components/MachinePicker";
import { FormSelect } from "@/components/ticket/FormSelect";
import { Button } from "@/components/ui/button";
import { useCreateTunnel, useFetchDnsZones, useProvisionTunnelAgent } from "@/hooks/DnsHooks";
import { useFetchMachines } from "@/hooks/MachineHooks";
import { useFetchProjects } from "@/hooks/ProjectHooks";
import { useWorkspaceStore } from "@/stores/workspaceStore";
import { soleItem, zoneAccounts, type CloudflareAccount, type TunnelDeployment } from "@/models/DNS";
import type { Project } from "@/models/Project";
import type { Machine } from "@/models/Machine";

// The account only has to be chosen when the token reaches more than one; with one it is preselected.
const tunnelDeploySchema = (chooseAccount: boolean, projectRequired: boolean) =>
  z.object({
    tunnel_name: z.string().trim().min(1, "Tunnel name is required"),
    account_id: chooseAccount ? z.string().min(1, "Choose the Cloudflare account") : z.string(),
    // Empty lets the server place the tunnel in the instance's default project (first-run setup has no workspace).
    project_id: projectRequired ? z.string().min(1, "Choose a project") : z.string(),
    target: z.string().trim().min(1, "Machine is required"),
    docker_network: z.string().trim().min(1, "Docker network is required"),
  });

type TunnelDeployFormData = z.infer<ReturnType<typeof tunnelDeploySchema>>;

interface TunnelDeployFieldsProps {
  projects: Project[];
  machines: Machine[];
  accounts: CloudflareAccount[];
  onConnected: (deployment: TunnelDeployment) => void;
}

// Mounted only once the lists are loaded so defaultValues can preselect the sole project, machine and account.
const TunnelDeployFields = ({ projects, machines, accounts, onConnected }: TunnelDeployFieldsProps) => {
  const createTunnel = useCreateTunnel();
  const provisionAgent = useProvisionTunnelAgent();
  const [pending, setPending] = useState<TunnelDeployment | null>(null);
  const form = useForm<TunnelDeployFormData>({
    defaultValues: {
      tunnel_name: "instance",
      account_id: soleItem(accounts)?.id ?? "",
      project_id: soleItem(projects)?.id ?? "",
      target: soleItem(machines)?.name ?? "",
      docker_network: "nexul_default",
    },
    resolver: zodResolver(tunnelDeploySchema(accounts.length > 1, projects.length > 0)),
  });
  const busy = createTunnel.isPending || provisionAgent.isPending;

  const onSubmit = async (data: TunnelDeployFormData) => {
    try {
      const tunnel = await createTunnel.mutateAsync({ name: data.tunnel_name, account_id: data.account_id });
      const agent = await provisionAgent.mutateAsync({
        tunnel_id: tunnel.id,
        project_id: data.project_id,
        target: data.target,
        docker_network: data.docker_network,
      });
      setPending({
        tunnelId: tunnel.id,
        tunnelName: tunnel.name,
        accountId: tunnel.account_id,
        serviceId: agent.service_id,
        target: data.target,
      });
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
          {accounts.length > 1 && (
            <div className="space-y-1.5">
              <FormSelect
                control={form.control}
                name="account_id"
                label="Cloudflare account"
                placeholder="Choose an account…"
                options={accounts.map((a) => ({ value: a.id, label: a.name }))}
              />
              <p className="text-xs text-muted-foreground">
                The account that owns the domain this instance will use. A tunnel only serves its own account's domains.
              </p>
            </div>
          )}
          <p className="font-mono text-xs text-muted-foreground">
            cloudflared-{form.watch("tunnel_name") || "instance"} on {form.watch("target") || "machine"}
          </p>
          <AdvancedFields>
            {projects.length > 0 && (
              <FormSelect
                control={form.control}
                name="project_id"
                label="Project"
                placeholder="Choose a project…"
                options={projects.map((p) => ({ value: p.id, label: p.name }))}
              />
            )}
            <MachinePicker control={form.control} name="target" />
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
  // Before sign-in there is no workspace to list projects from; the server then picks the instance's default.
  const inWorkspace = useWorkspaceStore((s) => s.selectedWorkspaceId) !== "";
  const { data: fetchedProjects, isPending: projectsFetching, error: projectsError } = useFetchProjects(inWorkspace);
  const projects = inWorkspace ? fetchedProjects : [];
  const projectsPending = inWorkspace && projectsFetching;
  const { data: machines, isPending: machinesPending, error: machinesError } = useFetchMachines();
  const { data: zones, isPending: zonesPending, error: zonesError } = useFetchDnsZones(true);

  return (
    <>
      {(projectsPending || machinesPending || zonesPending) && <LoadingDisplay />}
      {projectsError && <ErrorDisplay error={projectsError} />}
      {machinesError && <ErrorDisplay error={machinesError} />}
      {zonesError && <ErrorDisplay error={zonesError} />}
      {projects && machines && zones && (
        <TunnelDeployFields
          projects={projects}
          machines={machines}
          accounts={zoneAccounts(zones)}
          onConnected={onConnected}
        />
      )}
    </>
  );
};
