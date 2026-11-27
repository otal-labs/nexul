import { SlidersHorizontalIcon } from "lucide-react";
import { Fragment } from "react";

import { DrillChevron } from "@/components/computers/DrillChevron";
import { ListPaneGroup } from "@/components/listpane/ListPaneGroup";
import { ListPaneRow } from "@/components/listpane/ListPaneRow";
import { ProjectMark } from "@/components/project/ProjectMark";
import { useFetchPairingDefaults } from "@/hooks/PairingHooks";
import { useDefaultsSummary, useFetchProjectLinks, useProjectLinkSummary } from "@/hooks/PairingProjectHooks";
import { useFetchWorkspaceProjects } from "@/hooks/ProjectHooks";
import { computersSearch, DEFAULTS_ID, projectTarget } from "@/models/Agent";
import type { Project } from "@/models/Project";
import { useWorkspaceStore } from "@/stores/workspaceStore";

// What the run settings use, up to two lines: the model and where threads start are the point of it.
// It breaks only at its " · " separators, so "New worktree" is never split across two lines.
const Preview = ({ children }: { children: string | undefined }) => (
  <span className="line-clamp-2 min-w-0">
    {(children ?? "").split(" · ").map((part, i) => (
      <Fragment key={part}>
        {i > 0 && " "}
        <span className="inline-block max-w-full truncate align-bottom whitespace-nowrap">
          {i > 0 && "· "}
          {part}
        </span>
      </Fragment>
    ))}
  </span>
);

interface DefaultsRowProps {
  selected: boolean;
}

const DefaultsRow = ({ selected }: DefaultsRowProps) => {
  const { data: defaults, error } = useFetchPairingDefaults();
  const summary = useDefaultsSummary(defaults);

  return (
    <ListPaneRow
      to={computersSearch(DEFAULTS_ID)}
      title="Defaults"
      leading={<SlidersHorizontalIcon aria-hidden className="size-3.5 text-muted-foreground" />}
      detail={<Preview>{error ? "Couldn't load" : summary}</Preview>}
      trailing={<DrillChevron />}
      selected={selected}
    />
  );
};

interface ProjectRowProps {
  project: Project;
  selected: boolean;
}

const ProjectRow = ({ project, selected }: ProjectRowProps) => {
  const { data: links } = useFetchProjectLinks();
  const summary = useProjectLinkSummary(links?.find((l) => l.project_id === project.id));

  return (
    <ListPaneRow
      to={computersSearch(projectTarget(project.id))}
      title={project.name}
      leading={<ProjectMark project={project} className="size-6 rounded-md text-micro shadow-none" />}
      detail={<Preview>{summary}</Preview>}
      trailing={<DrillChevron />}
      selected={selected}
    />
  );
};

interface RunSettingsGroupProps {
  current: string;
}

// Where runs go: Defaults first, which every project uses until it changes something, then each project.
export const RunSettingsGroup = ({ current }: RunSettingsGroupProps) => {
  const workspaceId = useWorkspaceStore((s) => s.selectedWorkspaceId);
  const { data: projects } = useFetchWorkspaceProjects(workspaceId);

  return (
    <ListPaneGroup plain label="Where runs go" count={(projects?.length ?? 0) + 1}>
      <DefaultsRow selected={current === DEFAULTS_ID} />
      {projects?.map((project) => <ProjectRow key={project.id} project={project} selected={current === projectTarget(project.id)} />)}
    </ListPaneGroup>
  );
};
