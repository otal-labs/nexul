import { useEffect } from "react";
import { useSearchParams } from "react-router";

import { PageHeader } from "@/components/PageHeader";
import { InterviewPrototypeSwitcher } from "@/components/memory/prototype/InterviewPrototypeSwitcher";
import { InterviewPrototypeVariantA } from "@/components/memory/prototype/InterviewPrototypeVariantA";
import { InterviewPrototypeVariantB } from "@/components/memory/prototype/InterviewPrototypeVariantB";
import { InterviewPrototypeVariantC } from "@/components/memory/prototype/InterviewPrototypeVariantC";
import { STATES, VARIANTS, useInterviewPrototypeStore } from "@/components/memory/prototype/InterviewPrototypeStore";
import type { Project } from "@/models/Project";

// Throwaway: the Interview page's new flow, mocked in memory, in three layouts picked by ?variant= and ?state=.
export const InterviewPrototype = ({ project }: { project: Project }) => {
  const [params] = useSearchParams();
  const variant = Math.max(0, VARIANTS.findIndex((v) => v.key === params.get("variant")?.toUpperCase()));
  const state = Math.min(STATES.length, Math.max(1, Number(params.get("state")) || 1));
  const reset = useInterviewPrototypeStore((s) => s.reset);
  const epoch = useInterviewPrototypeStore((s) => s.epoch);

  useEffect(() => reset(state), [reset, state, variant]);

  return (
    <div className="pb-4">
      <PageHeader
        eyebrow={project.name}
        title="Interview"
        subtitle="Answer a few questions about how this project works. The agent asks about any gaps, then writes the rules every agent turn here follows."
      />
      <div key={`${variant}:${epoch}`} className="mt-12">
        {variant === 0 && <InterviewPrototypeVariantA />}
        {variant === 1 && <InterviewPrototypeVariantB />}
        {variant === 2 && <InterviewPrototypeVariantC />}
      </div>
      {import.meta.env.DEV && <InterviewPrototypeSwitcher variant={variant} state={state} />}
    </div>
  );
};
