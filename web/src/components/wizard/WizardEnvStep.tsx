import { useForm } from "react-hook-form";
import { useShallow } from "zustand/react/shallow";

import { ErrorDisplay } from "@/components/ErrorDisplay";
import { FormInput } from "@/components/FormInput";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { Button } from "@/components/ui/button";
import { useDeployStack, useFetchStack, useUpdateStackEnv } from "@/hooks/StackHooks";
import { useProjectWizardStore } from "@/stores/projectWizardStore";

interface WizardEnvStepProps {
  onDone: () => void;
}

// Only rendered when the scan found .env.example keys (spec §5/§7); values are optional, so an empty key is
// simply not written into the stack's env map. The service step held the first deploy back for this rung, so
// saving is what starts it: the values have to exist before compose reads `env_file: .env` or `${VAR}`.
export const WizardEnvStep = ({ onDone }: WizardEnvStepProps) => {
  const { stackId, scanResult, envValues } = useProjectWizardStore(
    useShallow((s) => ({ stackId: s.stackId, scanResult: s.scanResult, envValues: s.envValues })),
  );
  const setEnvValues = useProjectWizardStore((s) => s.setEnvValues);
  const { data: stack, isPending, error } = useFetchStack(stackId ?? undefined);
  const updateEnv = useUpdateStackEnv();
  const deployStack = useDeployStack();
  const envKeys = scanResult?.env_keys ?? [];
  // Every key needs a default (even "") — an RHF field left out of defaultValues renders uncontrolled, then
  // flips to controlled the moment it's typed into, which React (rightly) warns about.
  const form = useForm<Record<string, string>>({
    defaultValues: Object.fromEntries(envKeys.map((key) => [key, envValues[key] ?? ""])),
  });

  if (!stack) {
    return (
      <>
        {isPending && <LoadingDisplay />}
        {error && <ErrorDisplay error={error} title="Could not load the stack" />}
      </>
    );
  }

  const onSubmit = async (data: Record<string, string>) => {
    const filled = Object.fromEntries(Object.entries(data).filter(([, value]) => value.trim() !== ""));
    try {
      await updateEnv.mutateAsync({ ...stack, env: { ...stack.env, ...filled } });
      setEnvValues(filled);
      await deployStack.mutateAsync({ stackId: stack.id, ref: scanResult?.default_branch ?? "" });
      onDone();
    } catch {
      // Errors surface through the hook's toast; saving is retry-safe.
    }
  };

  return (
    <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-5">
      {envKeys.map((key) => (
        <FormInput key={key} control={form.control} name={key} label={key} placeholder="optional" />
      ))}
      <Button type="submit" className="w-full sm:w-auto" disabled={updateEnv.isPending || deployStack.isPending}>
        {updateEnv.isPending || deployStack.isPending ? "Saving…" : "Save & deploy"}
      </Button>
    </form>
  );
};
