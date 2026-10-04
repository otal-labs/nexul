import { useEffect } from "react";
import { useSearchParams } from "react-router";

import { ClarifyPrototypeSwitcher } from "@/components/doc/prototype/ClarifyPrototypeSwitcher";
import { ClarifyPrototypeVariantA } from "@/components/doc/prototype/ClarifyPrototypeVariantA";
import { ClarifyPrototypeVariantB } from "@/components/doc/prototype/ClarifyPrototypeVariantB";
import { ClarifyPrototypeVariantC } from "@/components/doc/prototype/ClarifyPrototypeVariantC";
import { STATES, VARIANTS, useClarifyDev, useClarifyPrototypeStore } from "@/components/doc/prototype/ClarifyPrototypeStore";
import type { Doc } from "@/models/Doc";

// Throwaway: a doc's clarification mocked in memory on the real doc route, three placements picked by ?variant=, ?state=, ?as=.
export const ClarifyPrototype = ({ doc }: { doc: Doc }) => {
  const [params] = useSearchParams();
  const variant = Math.max(0, VARIANTS.findIndex((v) => v.key === params.get("variant")?.toUpperCase()));
  const state = Math.min(STATES.length, Math.max(1, Number(params.get("state")) || 1));
  const dev = useClarifyDev();
  const reset = useClarifyPrototypeStore((s) => s.reset);
  const epoch = useClarifyPrototypeStore((s) => s.epoch);

  useEffect(() => reset(state), [reset, state, variant]);

  return (
    <div className="pb-20">
      <div key={`${variant}:${epoch}`} className="animate-in fade-in-0 duration-200 ease-out">
        {variant === 0 && <ClarifyPrototypeVariantA doc={doc} />}
        {variant === 1 && <ClarifyPrototypeVariantB doc={doc} />}
        {variant === 2 && <ClarifyPrototypeVariantC doc={doc} />}
      </div>
      {!import.meta.env.PROD && <ClarifyPrototypeSwitcher variant={variant} state={state} dev={dev} />}
    </div>
  );
};
