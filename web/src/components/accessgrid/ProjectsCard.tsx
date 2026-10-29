import { useState, type ReactNode } from "react";
import { Search } from "lucide-react";

import { Input } from "@/components/ui/input";
import { AccessGridTable } from "@/components/accessgrid/AccessGridTable";
import { PeopleCell } from "@/components/accessgrid/PeopleCell";
import { EmptyRow } from "@/components/EmptyRow";
import { ProjectMark } from "@/components/project/ProjectMark";
import { SettingsCard } from "@/components/settings/SettingsCard";
import { useAccessDraft } from "@/hooks/useAccessDraft";
import type { AccessGrid } from "@/models/AccessGrid";

interface ProjectsCardProps {
  grid: AccessGrid;
  footer: ReactNode;
  aside?: ReactNode;
  fill?: boolean;
}

const SEARCH_FROM = 10;

// Projects by roles, with the people a project names beyond its roles in a trailing column.
export const ProjectsCard = ({ grid, footer, aside, fill = false }: ProjectsCardProps) => {
  const [query, setQuery] = useState("");
  const draft = useAccessDraft(grid);
  const needle = query.trim().toLowerCase();
  const rows = grid.projects
    .filter((project) => !needle || project.name.toLowerCase().includes(needle))
    .map((project) => {
      const { userIds } = draft.listing(project.id);
      return {
        id: project.id,
        name: project.name,
        mark: <ProjectMark project={project} className="size-6 rounded-md text-[9px]" />,
        trailing: <PeopleCell projectName={project.name} userIds={userIds} onToggle={(userId) => draft.togglePerson(project.id, userId)} />,
      };
    });

  return (
    <SettingsCard
      id="projects"
      title="Projects"
      description="Tick the roles that belong in each project. Everyone with a ticked role sees the project and everything in it."
      aside={aside}
      footer={footer}
      fill={fill}
    >
      {grid.projects.length > SEARCH_FROM && (
        <label className="relative mb-4 block max-w-xs shrink-0">
          <Search className="pointer-events-none absolute top-1/2 left-2.5 size-4 -translate-y-1/2 text-muted-foreground" aria-hidden />
          <Input value={query} onChange={(event) => setQuery(event.target.value)} placeholder={`Filter ${grid.projects.length} projects`} aria-label="Filter projects" className="h-8 pl-8" />
        </label>
      )}
      {grid.projects.length === 0 && <EmptyRow flush>No projects yet.</EmptyRow>}
      {grid.projects.length > 0 && rows.length === 0 && <EmptyRow flush>No project matches “{query}”.</EmptyRow>}
      {rows.length > 0 && (
        <AccessGridTable
          label="Project"
          leadLabel={needle ? "Every project shown" : "Every project"}
          rows={rows}
          roles={grid.roles}
          marks="ticks"
          isOn={(rowId, roleId) => draft.listing(rowId).roleIds.includes(roleId)}
          isChanged={(rowId, roleId) => draft.roleChanged(rowId, roleId)}
          onToggle={draft.toggleRole}
          onSetAll={(roleId, on) => draft.setRoleOnProjects(roleId, rows.map((row) => row.id), on)}
          trailingHead="People"
          fill={fill}
        />
      )}
    </SettingsCard>
  );
};
