import { useCallback, useMemo } from "react";

import { useSetupDraftStore } from "@/stores/setupDraftStore";
import type { SetupModelChoice } from "@/models/Pairing";

// The Set up step's include switches: the unsaved flip, else what the computer has saved as skipped; a run sends only the included ones.
export const useSetupProviders = (computerId: string, choices: SetupModelChoice[], skipped: string[]) => {
  const flipped = useSetupDraftStore((s) => s.drafts[computerId]?.included);
  const edit = useSetupDraftStore((s) => s.edit);
  const included = useMemo(
    () => choices.filter((c) => flipped?.[c.provider] ?? !skipped.includes(c.provider)).map((c) => c.provider),
    [choices, flipped, skipped],
  );
  const excluded = useMemo(() => choices.map((c) => c.provider).filter((p) => !included.includes(p)), [choices, included]);
  const include = useCallback(
    (provider: string, on: boolean) => edit(computerId, (d) => ({ ...d, included: { ...d.included, [provider]: on } })),
    [computerId, edit],
  );
  return { included, excluded, include };
};
