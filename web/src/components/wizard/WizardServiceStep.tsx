import { zodResolver } from "@hookform/resolvers/zod";
import { useForm } from "react-hook-form";
import { z } from "zod";
import { useShallow } from "zustand/react/shallow";

import { ErrorDisplay } from "@/components/ErrorDisplay";
import { FormInput } from "@/components/FormInput";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { Button } from "@/components/ui/button";
import { CandidateChoice } from "@/components/wizard/CandidateChoice";
import { declaredFrom } from "@/components/wizard/declaredFrom";
import { MachinePicker } from "@/components/wizard/MachinePicker";
import { useCreateStack, useDeployStack, useFetchStack, useUpdateStack } from "@/hooks/StackHooks";
import type { Candidate, Repo } from "@/models/Repository";
import type { BuildSource, CreateStackInput } from "@/models/Stack";
import { useProjectWizardStore } from "@/stores/projectWizardStore";
import { slugify } from "@/utils/SlugUtility";

const ServiceFormSchema = z.object({
  name: z.string().trim().min(1, "Name is required"),
  machine: z.string().trim().min(1, "Machine is required"),
});
type ServiceFormData = z.infer<typeof ServiceFormSchema>;

// Shared by the create path (POST) and the attach path (PATCH) so the two build the exact same shape.
const buildSourceFrom = (repository: Repo, candidate: Candidate, branch: string): BuildSource => ({
  repo_owner: repository.owner,
  repo_name: repository.name,
  branch,
  ...(candidate.kind === "dockerfile" && { dockerfile: candidate.path }),
  ...(candidate.kind === "compose" && { compose_path: candidate.path }),
});

interface WizardServiceStepProps {
  onDone: () => void;
}

// Candidate pick (compose preselected by the repository step, switchable here for a monorepo scan) and the
// reachable service preview are shared by both modes this step can run in:
// - Create (door 1/2): also asks for name + machine, then creates the stack and starts its first deploy in one
//   call (deploy: true).
// - Attach (door 3, attachStackId set): the stack already has a name and machine, so this only PATCHes its
//   build_source (and compose_path) onto the existing record and starts the deploy itself.
// Both modes hold the first deploy back when the scan found .env.example keys — a compose file's
// `env_file: .env` or `${VAR}` needs the values in place first, which the env step supplies.
export const WizardServiceStep = ({ onDone }: WizardServiceStepProps) => {
  const { projectId, repository, scanResult, candidate, name, attachStackId } = useProjectWizardStore(
    useShallow((s) => ({
      projectId: s.projectId,
      repository: s.repository,
      scanResult: s.scanResult,
      candidate: s.candidate,
      name: s.name,
      attachStackId: s.attachStackId,
    })),
  );
  const envStepFollows = (scanResult?.env_keys.length ?? 0) > 0;
  const setCandidate = useProjectWizardStore((s) => s.setCandidate);
  const setName = useProjectWizardStore((s) => s.setName);
  const setMachine = useProjectWizardStore((s) => s.setMachine);
  const setStackId = useProjectWizardStore((s) => s.setStackId);
  const createStack = useCreateStack();
  const updateStack = useUpdateStack();
  const deployStack = useDeployStack();
  const {
    data: attachStack,
    isPending: isAttachPending,
    error: attachError,
  } = useFetchStack(attachStackId ?? undefined);

  const form = useForm<ServiceFormData>({
    defaultValues: { name: name || candidate?.name || "", machine: "" },
    resolver: zodResolver(ServiceFormSchema),
  });

  if (!candidate || !projectId) return null;

  const candidates = scanResult?.candidates ?? [candidate];
  const branch = scanResult?.default_branch || repository?.default_branch || "main";
  const candidatePicker = (
    <>
      {candidates.length > 1 && (
        <CandidateChoice candidates={candidates} value={candidate} onChange={setCandidate} />
      )}
      {candidate.reachable && (
        <p className="font-mono text-xs text-muted-foreground">
          Reachable: {candidate.reachable.service}:{candidate.reachable.port}
        </p>
      )}
    </>
  );

  if (attachStackId) {
    if (!attachStack) {
      return (
        <>
          {isAttachPending && <LoadingDisplay />}
          {attachError && <ErrorDisplay error={attachError} title="Could not load the stack" />}
        </>
      );
    }

    // Import adopts everything as compose, so the candidate's kind decides the strategy the same way create does;
    // a Dockerfile candidate turns the stack into a run stack, which needs a docker network like create gives it.
    const onAttach = async () => {
      const strategy = candidate.kind === "compose" ? "compose" : "run";
      try {
        const stack = await updateStack.mutateAsync({
          ...attachStack,
          strategy,
          ...(repository && { build_source: buildSourceFrom(repository, candidate, branch) }),
          ...(strategy === "compose" && { compose_path: candidate.path }),
          ...(strategy === "run" && { docker_network: attachStack.docker_network ?? `${attachStack.slug}_default` }),
        });
        setName(stack.name);
        setMachine(stack.machine);
        setStackId(stack.id);
        if (!envStepFollows) {
          await deployStack.mutateAsync({ stackId: stack.id, ref: scanResult?.default_branch ?? "" });
        }
        onDone();
      } catch {
        // Errors surface through the hooks' toasts; the PATCH and deploy are both retry-safe.
      }
    };

    return (
      <div className="space-y-5">
        {candidatePicker}
        <p className="text-sm text-muted-foreground">
          Attaching to <span className="font-medium text-foreground">{attachStack.name}</span> on{" "}
          <span className="font-mono text-foreground">{attachStack.machine}</span>.
        </p>
        <Button
          type="button"
          onClick={() => void onAttach()}
          className="w-full sm:w-auto"
          disabled={updateStack.isPending || deployStack.isPending}
        >
          {updateStack.isPending || deployStack.isPending ? "Attaching…" : "Attach"}
        </Button>
      </div>
    );
  }

  const onSubmit = async (data: ServiceFormData) => {
    const strategy = candidate.kind === "compose" ? "compose" : "run";
    const input: CreateStackInput = {
      project_id: projectId,
      name: data.name,
      machine: data.machine,
      strategy,
      ...(strategy === "compose" && { compose_path: candidate.path }),
      // ponytail: a run stack needs some docker network; slugified-name default until the wizard exposes one.
      ...(strategy === "run" && { docker_network: `${slugify(data.name)}_default` }),
      ...(repository && {
        build_source: buildSourceFrom(repository, candidate, branch),
        link_repository: true,
      }),
      declared: declaredFrom(candidate, data.name),
      deploy: !envStepFollows,
    };
    try {
      const stack = await createStack.mutateAsync(input);
      setName(data.name);
      setMachine(data.machine);
      setStackId(stack.id);
      onDone();
    } catch {
      // Errors surface through the hook's toast; creation is retry-safe.
    }
  };

  return (
    <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-5">
      {candidatePicker}
      <FormInput control={form.control} name="name" label="Name" placeholder="e.g. web" />
      <MachinePicker control={form.control} name="machine" />
      <Button type="submit" className="w-full sm:w-auto" disabled={createStack.isPending}>
        {createStack.isPending ? "Creating…" : envStepFollows ? "Create" : "Create & deploy"}
      </Button>
    </form>
  );
};
