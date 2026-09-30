import { useSetupFolder } from "@/hooks/useSetupFolder";
import { useSetupModels } from "@/hooks/useSetupModels";
import { useSetupProviders } from "@/hooks/useSetupProviders";
import type { ComputerSetup, SetupChoices } from "@/models/Pairing";

// Everything the Set up step shows and saves, for the step itself and the dialog's Done button alike.
export const useSetupChoices = (setup: ComputerSetup) => {
  const { choices, models, options, pick, pickOptions } = useSetupModels(setup.computer_id, setup);
  const { included, excluded, include } = useSetupProviders(setup.computer_id, choices, setup.skipped_providers);
  const { projects, folder, pick: pickFolder } = useSetupFolder(setup.computer_id, setup.folder);
  // A provider the harness does not list right now keeps what was saved for it.
  const body: SetupChoices = {
    skipped_providers: [...setup.skipped_providers.filter((p) => !choices.some((c) => c.provider === p)), ...excluded],
    models: { ...setup.models, ...models },
    model_options: { ...setup.model_options, ...options },
    folder,
  };
  return { choices, models, options, pick, pickOptions, included, excluded, include, projects, folder, pickFolder, body };
};
