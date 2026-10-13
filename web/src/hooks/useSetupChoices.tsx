import { useMemo } from "react";

import { useSaveSetupChoices } from "@/hooks/ComputerSetupHooks";
import { useSetupFolder } from "@/hooks/useSetupFolder";
import { useSetupModels } from "@/hooks/useSetupModels";
import type { ComputerSetup, SetupChoices } from "@/models/Pairing";

// Everything the Set up step shows and saves, for the step itself and the dialog's Done button alike.
export const useSetupChoices = (setup: ComputerSetup) => {
  const { choices, models, options, pick, pickOptions } = useSetupModels(setup.computer_id, setup);
  const { projects, folder, pick: pickFolder } = useSetupFolder(setup.computer_id, setup.folder);
  const save = useSaveSetupChoices(setup.computer_id);
  const included = useMemo(
    () => choices.map((c) => c.provider).filter((p) => !setup.skipped_providers.includes(p)),
    [choices, setup.skipped_providers],
  );
  const body: SetupChoices = {
    skipped_providers: setup.skipped_providers,
    models: { ...setup.models, ...models },
    model_options: { ...setup.model_options, ...options },
    folder,
  };
  // A switch saves at once: plays leave a switched-off provider out, so it can't wait for Done.
  const include = (provider: string, on: boolean) => {
    const others = setup.skipped_providers.filter((p) => p !== provider);
    save.mutate({ ...body, skipped_providers: on ? others : [...others, provider] });
  };
  return { choices, models, options, pick, pickOptions, included, include, projects, folder, pickFolder, body };
};
