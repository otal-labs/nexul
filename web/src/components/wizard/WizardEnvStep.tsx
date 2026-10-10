import { useMemo, useState } from "react";
import { useForm } from "react-hook-form";
import { useShallow } from "zustand/react/shallow";

import { ErrorDisplay } from "@/components/ErrorDisplay";
import { FormInput } from "@/components/FormInput";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { Button } from "@/components/ui/button";
import { EnvModeToggle, type EnvMode } from "@/components/wizard/EnvModeToggle";
import { EnvPasteField } from "@/components/wizard/EnvPasteField";
import { WizardFooter } from "@/components/wizard/WizardFooter";
import { WizardSkipLink } from "@/components/wizard/WizardSkipLink";
import { useDeployStack, useFetchStack, useUpdateStackEnv } from "@/hooks/StackHooks";
import { useWizardEnvKeys } from "@/hooks/useWizardSetup";
import { useProjectWizardStore } from "@/stores/projectWizardStore";
import type { Stack } from "@/models/Stack";
import { formatEnvFile, parseEnvFile } from "@/models/EnvFile";

interface WizardEnvStepProps {
  onDone: () => void;
  onBack?: (() => void) | undefined;
  onSkip?: (() => void) | undefined;
}

export const WizardEnvStep = (props: WizardEnvStepProps) => {
  const stackId = useProjectWizardStore((s) => s.stackId);
  const envKeys = useWizardEnvKeys();
  const { data: stack, isPending, error } = useFetchStack(stackId ?? undefined);
  return (
    <div>
      {isPending && <LoadingDisplay />}
      {error && <ErrorDisplay error={error} title="Couldn't load the stack." />}
      {stack && <WizardEnvForm key={stack.id} stack={stack} envKeys={envKeys} {...props} />}
    </div>
  );
};

interface WizardEnvFormProps extends WizardEnvStepProps {
  stack: Stack;
  envKeys: string[];
}

const WizardEnvForm = ({ stack, envKeys, onDone, onBack, onSkip }: WizardEnvFormProps) => {
  const { envValues } = useProjectWizardStore(
    useShallow((s) => ({ envValues: s.envValues })),
  );
  const initialValues = { ...stack.env, ...envValues };
  const setEnvValues = useProjectWizardStore((s) => s.setEnvValues);
  const updateEnv = useUpdateStackEnv();
  const deployStack = useDeployStack();
  // Every key needs a default (even "") — an RHF field left out of defaultValues renders uncontrolled, then
  // flips to controlled the moment it's typed into, which React (rightly) warns about.
  const form = useForm<Record<string, string>>({
    defaultValues: Object.fromEntries(envKeys.map((key) => [key, initialValues[key] ?? ""])),
  });
  const [mode, setMode] = useState<EnvMode>("fields");
  const [text, setText] = useState("");
  // Keys beyond the detected ones have no field; they ride along in Paste and are still saved.
  const [extras, setExtras] = useState(() => Object.fromEntries(Object.entries(initialValues).filter(([key]) => !envKeys.includes(key))));
  const parsed = useMemo(() => parseEnvFile(text), [text]);
  const pasteBlocked = mode === "paste" && parsed.invalidLines.length > 0;

  const changeMode = (next: EnvMode) => {
    if (next === "paste") setText(formatEnvFile({ ...extras, ...form.getValues() }, [...envKeys, ...Object.keys(extras)]));
    if (next === "fields") {
      form.reset(Object.fromEntries(envKeys.map((key) => [key, parsed.values[key] ?? ""])));
      setExtras(Object.fromEntries(Object.entries(parsed.values).filter(([key]) => !envKeys.includes(key))));
    }
    setMode(next);
  };

  const onSubmit = async (data: Record<string, string>) => {
    if (pasteBlocked) return;
    const entered = mode === "paste" ? parsed.values : { ...extras, ...data };
    const filled = Object.fromEntries(Object.entries(entered).filter(([, value]) => value.trim() !== ""));
    try {
      await updateEnv.mutateAsync({ ...stack, env: { ...stack.env, ...filled } });
      setEnvValues(filled);
      await deployStack.mutateAsync({ stackId: stack.id, ref: stack.build_source?.branch ?? "" });
      onDone();
    } catch {
      // Errors surface through the hook's toast; saving is retry-safe.
    }
  };

  return (
    <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-5">
      <div className="flex">
        <EnvModeToggle mode={mode} onMode={changeMode} fieldsDisabled={pasteBlocked} />
      </div>
      {mode === "fields" &&
        envKeys.map((key) => <FormInput key={key} control={form.control} name={key} label={key} placeholder="optional" />)}
      {mode === "fields" && Object.keys(extras).length > 0 && (
        <p className="text-xs text-muted-foreground">Also saving: {Object.keys(extras).join(", ")}</p>
      )}
      {mode === "paste" && <EnvPasteField text={text} onText={setText} parsed={parsed} />}
      <WizardFooter onBack={onBack} skip={onSkip && <WizardSkipLink onClick={onSkip} />}>
        <Button type="submit" disabled={pasteBlocked} loading={updateEnv.isPending || deployStack.isPending}>
          Save & deploy
        </Button>
      </WizardFooter>
    </form>
  );
};
