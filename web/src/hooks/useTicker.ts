import { useState } from "react";

import { errorMessage } from "@/api/client";
import type { CheckOutcome } from "@/components/TickerRow";
import type { CredentialCheck } from "@/models/Connectors";

type Fields = Record<string, string>;

// The ticker: Verify fans one request per check out in parallel and ticks each TickerRow as it lands; Continue unlocks once
// every required check is green, while a failed advisory check only warns.
// Derived, not an effect: the rows stay lit only while every verified field still matches what the provider accepted.
export const useTicker = (
  checks: CredentialCheck[],
  current: Record<string, string | undefined>,
  run: (key: string, data: Fields) => Promise<unknown>,
) => {
  const [verifiedFor, setVerifiedFor] = useState<Fields | null>(null);
  const [outcomes, setOutcomes] = useState<Record<string, CheckOutcome>>({});
  const fresh =
    verifiedFor !== null && Object.entries(verifiedFor).every(([key, value]) => value === current[key]?.trim());
  const verified =
    fresh && checks.every((c) => outcomes[c.key]?.state === "ok" || outcomes[c.key]?.state === "warning");
  const verifying = fresh && checks.some((c) => outcomes[c.key]?.state === "pending");

  const runCheck = async ({ key, advisory }: CredentialCheck, data: Fields) => {
    try {
      await run(key, data);
      setOutcomes((prev) => ({ ...prev, [key]: { state: "ok" } }));
    } catch (err) {
      const state = advisory ? "warning" : "failed";
      setOutcomes((prev) => ({ ...prev, [key]: { state, message: errorMessage(err) } }));
    }
  };

  const verify = async (data: Fields) => {
    setVerifiedFor(data);
    setOutcomes(Object.fromEntries(checks.map((c) => [c.key, { state: "pending" }])));
    await Promise.all(checks.map((c) => runCheck(c, data)));
  };

  const reset = () => {
    setVerifiedFor(null);
    setOutcomes({});
  };

  const outcomeFor = (key: string): CheckOutcome => (fresh ? (outcomes[key] ?? { state: "pending" }) : { state: "idle" });

  return { fresh, verified, verifying, verify, reset, outcomeFor };
};
