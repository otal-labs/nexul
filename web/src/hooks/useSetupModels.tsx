import { useCallback, useMemo, useState } from "react";

import { useFetchHarnessProviders, useFetchPairingDefaults } from "@/hooks/PairingHooks";
import { setupModelChoices } from "@/models/Pairing";

// The Set up step's model per provider: the user's pick, else the preselection; Start and Retry both send it.
export const useSetupModels = (computerId: string) => {
  const { data: providers } = useFetchHarnessProviders(computerId);
  const { data: defaults } = useFetchPairingDefaults();
  const [picked, setPicked] = useState<Record<string, string>>({});
  const choices = useMemo(() => setupModelChoices(providers ?? [], defaults), [providers, defaults]);
  const models = useMemo(
    () => Object.fromEntries(choices.map((c) => [c.provider, picked[c.provider] ?? c.preselected])),
    [choices, picked],
  );
  const pick = useCallback((provider: string, model: string) => setPicked((p) => ({ ...p, [provider]: model })), []);
  return { choices, models, pick };
};
