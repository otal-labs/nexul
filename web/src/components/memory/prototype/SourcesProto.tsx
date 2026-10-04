import { useEffect } from "react";
import { useSearchParams } from "react-router";

import { PageHeader } from "@/components/PageHeader";
import { SourcesProtoSwitcher } from "@/components/memory/prototype/SourcesProtoSwitcher";
import { SourcesProtoVariantA } from "@/components/memory/prototype/SourcesProtoVariantA";
import { SourcesProtoVariantB } from "@/components/memory/prototype/SourcesProtoVariantB";
import { SourcesProtoVariantC } from "@/components/memory/prototype/SourcesProtoVariantC";
import { STATES, VARIANTS, useSourcesProtoStore } from "@/components/memory/prototype/SourcesProtoStore";
import type { Project } from "@/models/Project";

// Throwaway: the Interview page with sources and drafts, mocked in memory, in three layouts picked by ?variant= and ?state=.
export const SourcesProto = ({ project }: { project: Project }) => {
  const [params] = useSearchParams();
  const variant = Math.max(0, VARIANTS.findIndex((v) => v.key === params.get("variant")?.toUpperCase()));
  const state = Math.min(STATES.length, Math.max(1, Number(params.get("state")) || 1));
  const reset = useSourcesProtoStore((s) => s.reset);
  const epoch = useSourcesProtoStore((s) => s.epoch);

  useEffect(() => reset(state), [reset, state, variant]);

  return (
    <div className="space-y-10 pb-4">
      <PageHeader
        eyebrow={project.name}
        title="Interview"
        subtitle="Answer a few questions about how this project works. The agent asks about any gaps, then writes the rules every agent turn here follows."
      />
      <div key={`${variant}:${epoch}`}>
        {variant === 0 && <SourcesProtoVariantA />}
        {variant === 1 && <SourcesProtoVariantB />}
        {variant === 2 && <SourcesProtoVariantC />}
      </div>
      {import.meta.env.DEV && <SourcesProtoSwitcher variant={variant} state={state} />}
    </div>
  );
};
