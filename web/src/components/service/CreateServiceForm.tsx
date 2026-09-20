import { useFormDialogContext } from "@/components/dialogs/FormDialogContext";
import { FormInput } from "@/components/FormInput";
import { RunnerPicker } from "@/components/RunnerPicker";
import { ServiceFormAdaptor } from "@/components/service/ServiceFormAdaptor";
import { FormSelect } from "@/components/ticket/FormSelect";
import { FormTextarea } from "@/components/ticket/FormTextarea";
import { useCreateService, useDeployService } from "@/hooks/ServiceHooks";
import { parseEnv } from "@/lib/env";
import { DeployStrategy, type ServiceDef, type ServiceFormData } from "@/models/Service";

interface CreateServiceFormProps {
  projectId: string;
}

export const emptyServiceForm = (): ServiceFormData => ({
  name: "",
  target: "",
  strategy: "compose",
  compose_dir: "",
  docker_network: "",
  health_url: "",
  env: "",
  image: "",
  ref: "",
  build_repo_owner: "",
  build_repo_name: "",
  build_branch: "",
  build_dockerfile: "",
  build_compose_path: "",
});

const buildCreatePayload = (projectId: string, formData: ServiceFormData): Partial<ServiceDef> => {
  const env = parseEnv(formData.env);
  const payload: Record<string, unknown> = {
    project_id: projectId,
    name: formData.name,
    target: formData.target,
    strategy: formData.strategy,
    health_check: { url: formData.health_url },
  };
  if (formData.strategy === "compose") payload.compose_dir = formData.compose_dir;
  if (formData.strategy === "run") payload.docker_network = formData.docker_network;
  if (Object.keys(env).length) payload.env = env;
  if (formData.build_repo_owner && formData.build_repo_name) {
    payload.build_source = {
      repo_owner: formData.build_repo_owner,
      repo_name: formData.build_repo_name,
      branch: formData.build_branch || undefined,
      dockerfile: formData.build_dockerfile || undefined,
      compose_path: formData.build_compose_path || undefined,
    };
  }
  return payload as Partial<ServiceDef>;
};

export const CreateServiceForm = ({ projectId }: CreateServiceFormProps) => {
  const { control, onSubmit } = useFormDialogContext<ServiceFormData>();
  const createService = useCreateService();
  const deploy = useDeployService();

  onSubmit(async (formData) => {
    if (!projectId) throw new Error("Choose a project first");
    const svc = await createService.mutateAsync(buildCreatePayload(projectId, formData));
    // Creating a service triggers its first deploy.
    if (formData.image || formData.ref) {
      await deploy.mutateAsync({
        serviceId: svc.id,
        ...(formData.image ? { image: formData.image } : {}),
        ...(formData.ref ? { ref: formData.ref } : {}),
      });
    }
    return { id: svc.id, ...formData };
  });

  return (
    <div className="space-y-4">
      <FormInput control={control} name="name" label="Name" placeholder="api" />
      <RunnerPicker control={control} name="target" />

      <FormSelect
        control={control}
        name="strategy"
        label="Strategy"
        options={[
          { value: DeployStrategy.Compose, label: "Compose" },
          { value: DeployStrategy.Run, label: "Docker run" },
        ]}
      />

      <ServiceFormAdaptor />

      <FormInput
        control={control}
        name="health_url"
        label="Health check URL"
        placeholder="http://10.0.0.1:8080/health"
      />

      <FormTextarea
        control={control}
        name="env"
        label="Environment (KEY=value per line)"
        className="font-mono text-xs"
        rows={3}
        placeholder="PORT=8080"
      />

      <fieldset className="rounded-lg border border-border p-3">
        <legend className="px-1 text-sm font-medium">First deploy</legend>
        <div className="grid gap-3">
          <FormInput
            control={control}
            name="image"
            label="Image (pre-built)"
            placeholder="ghcr.io/acme/api:v1"
          />
          <p className="text-xs text-muted-foreground">or</p>
          <FormInput
            control={control}
            name="ref"
            label="Ref to build (repo-driven)"
            placeholder="main"
          />
        </div>
      </fieldset>

      <fieldset className="rounded-lg border border-border p-3">
        <legend className="px-1 text-sm font-medium">Build source (repo-driven)</legend>
        <p className="mb-3 text-xs text-muted-foreground">
          Optional. When set, the runner clones this repository and builds from it
          (build &amp; deploy actions). Required before a ref deploy can trigger.
        </p>
        <div className="grid gap-3">
          <FormInput control={control} name="build_repo_owner" label="Repo owner" placeholder="acme" />
          <FormInput control={control} name="build_repo_name" label="Repo name" placeholder="api" />
          <FormInput control={control} name="build_branch" label="Default branch" placeholder="main" />
          <FormInput
            control={control}
            name="build_dockerfile"
            label="Dockerfile path (run strategy)"
            placeholder="Dockerfile"
          />
          <FormInput
            control={control}
            name="build_compose_path"
            label="Compose file path (compose strategy)"
            placeholder="docker-compose.yml"
          />
        </div>
      </fieldset>
    </div>
  );
};
