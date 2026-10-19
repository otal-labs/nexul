import { PlayFact } from "@/components/play/PlayFact";
import { PlayTemplateLine } from "@/components/play/PlayTemplateLine";
import { useFetchProjects } from "@/hooks/ProjectHooks";
import { PLAY_STAGE_LABELS, type Play } from "@/models/Play";

interface PlayFactsProps {
  play: Play;
}

// What the play is, read only; the Edit button changes it.
export const PlayFacts = ({ play }: PlayFactsProps) => {
  const { data: projects } = useFetchProjects();
  const excluded = (projects ?? []).filter((project) => play.excluded_project_ids.includes(project.id)).map((project) => project.name);

  return (
    <div className="max-w-3xl space-y-8">
      {play.type === "ticket" && play.show_when_stage && (
        <PlayFact label="Shows on">
          <p className="text-sm">{PLAY_STAGE_LABELS[play.show_when_stage]}</p>
        </PlayFact>
      )}
      <PlayFact label="Excluded projects">
        <p className="text-sm">{excluded.length > 0 ? excluded.join(", ") : "None"}</p>
      </PlayFact>
      {play.builtin_key && <PlayTemplateLine play={play} />}
      <PlayFact label="Instructions">
        {play.instructions !== "" && (
          <p className="rounded-md bg-muted/50 p-4 text-sm leading-relaxed whitespace-pre-wrap [overflow-wrap:anywhere]">{play.instructions}</p>
        )}
        {play.instructions === "" && <p className="text-sm text-muted-foreground">No instructions</p>}
      </PlayFact>
    </div>
  );
};
