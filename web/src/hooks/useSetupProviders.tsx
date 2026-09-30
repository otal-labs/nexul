import { useCallback, useMemo, useState } from "react";

import type { SetupModelChoice } from "@/models/Pairing";

// The Set up step's include switches: the user's flip, else what the computer's last run skipped; a run sends only the included ones.
export const useSetupProviders = (choices: SetupModelChoice[], skipped: string[]) => {
  const [flipped, setFlipped] = useState<Record<string, boolean>>({});
  const included = useMemo(
    () => choices.filter((c) => flipped[c.provider] ?? !skipped.includes(c.provider)).map((c) => c.provider),
    [choices, flipped, skipped],
  );
  const excluded = useMemo(() => choices.map((c) => c.provider).filter((p) => !included.includes(p)), [choices, included]);
  const include = useCallback((provider: string, on: boolean) => setFlipped((f) => ({ ...f, [provider]: on })), []);
  return { included, excluded, include };
};
