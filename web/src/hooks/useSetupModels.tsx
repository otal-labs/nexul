import { useCallback, useMemo } from "react";

import { useFetchHarnessProviders, useFetchPairingDefaults } from "@/hooks/PairingHooks";
import { useSetupDraftStore } from "@/stores/setupDraftStore";
import { setupModelChoices, type OptionSetting, type SetupChoices } from "@/models/Pairing";

// The Set up step's model and its options per provider: the unsaved edit, else the saved pick, else the preselection.
export const useSetupModels = (computerId: string, saved: SetupChoices) => {
  const { data: providers } = useFetchHarnessProviders(computerId);
  const { data: defaults } = useFetchPairingDefaults();
  const draft = useSetupDraftStore((s) => s.drafts[computerId]);
  const edit = useSetupDraftStore((s) => s.edit);
  const choices = useMemo(() => setupModelChoices(providers ?? [], defaults, saved), [providers, defaults, saved]);
  const models = useMemo(
    () => Object.fromEntries(choices.map((c) => [c.provider, draft?.models?.[c.provider] ?? c.preselected])),
    [choices, draft],
  );
  const options = useMemo(
    () => Object.fromEntries(choices.map((c) => [c.provider, draft?.options?.[c.provider] ?? c.preselectedOptions])),
    [choices, draft],
  );
  // Options only mean something for their model, so a new model starts on its defaults.
  const pick = useCallback(
    (provider: string, model: string) =>
      edit(computerId, (d) => ({ ...d, models: { ...d.models, [provider]: model }, options: { ...d.options, [provider]: [] } })),
    [computerId, edit],
  );
  const pickOptions = useCallback(
    (provider: string, next: OptionSetting[]) => edit(computerId, (d) => ({ ...d, options: { ...d.options, [provider]: next } })),
    [computerId, edit],
  );
  return { choices, models, options, pick, pickOptions };
};
