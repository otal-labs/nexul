import type { ReactNode } from "react";

import { Microheader } from "@/components/Microheader";
import { PlayTemplateLine } from "@/components/play/PlayTemplateLine";
import { useFetchProjects } from "@/hooks/ProjectHooks";
import { PLAY_STAGE_LABELS, type Play } from "@/models/Play";

interface PlayFactsStripProps {
  play: Play;
}

const Fact = ({ label, children }: { label: string; children: ReactNode }) => (
  <div className="min-w-0 space-y-1.5">
    <dt>
      <Microheader>{label}</Microheader>
    </dt>
    <dd className="text-[13px] text-foreground [&>div]:text-[13px] [&>div]:leading-[inherit]">{children}</dd>
  </div>
);

// What the play is, read only, in one row of label over value; the Edit button changes it.
export const PlayFactsStrip = ({ play }: PlayFactsStripProps) => {
  const { data: projects } = useFetchProjects();
  const excluded = (projects ?? []).filter((project) => play.excluded_project_ids.includes(project.id)).map((project) => project.name);

  return (
    <dl className="flex flex-wrap gap-x-12 gap-y-4">
      {play.type === "ticket" && play.show_when_stage && <Fact label="Shows on">{PLAY_STAGE_LABELS[play.show_when_stage]}</Fact>}
      <Fact label="Excluded">
        {excluded.length === 0 && <span className="text-muted-foreground">None</span>}
        {excluded.length > 0 && (
          <span className="flex flex-wrap gap-1.5">
            {excluded.map((name) => (
              <span key={name} className="rounded-full border border-border bg-foreground/[0.04] px-2 py-0.5 text-xs text-muted-foreground">
                {name}
              </span>
            ))}
          </span>
        )}
      </Fact>
      {play.builtin_key && (
        <Fact label="Template">
          <PlayTemplateLine play={play} />
        </Fact>
      )}
    </dl>
  );
};
