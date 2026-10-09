import { useFetchProjectDeleteImpact } from "@/hooks/ProjectHooks";
import type { Project } from "@/models/Project";

interface ProjectSettingsMetaProps {
  project: Project;
}

const count = (n: number, word: string) => `${n} ${word}${n === 1 ? "" : "s"}`;

// The settings header's facts: the prefix, then what the project holds, from the same counts its danger zone checks.
export const ProjectSettingsMeta = ({ project }: ProjectSettingsMetaProps) => {
  const { data: impact } = useFetchProjectDeleteImpact(project.id);
  const empty = impact && impact.tickets + impact.repos + impact.services === 0;
  return (
    <span className="font-mono text-xs tabular-nums">
      {project.prefix}
      {empty && " · empty"}
      {impact && !empty && ` · ${count(impact.tickets, "ticket")} · ${count(impact.repos, "repo")} · ${count(impact.services, "service")}`}
    </span>
  );
};
