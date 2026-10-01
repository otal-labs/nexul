import { useState } from "react";
import { SearchIcon } from "lucide-react";

import { Microheader } from "@/components/access/Microheader";
import { ProjectAccessRow } from "@/components/access/ProjectAccessRow";
import { EmptyRow } from "@/components/EmptyRow";
import { Input } from "@/components/ui/input";

export interface AccessProject {
  id: string;
  name: string;
}

interface ProjectAccessListProps {
  projects: AccessProject[];
  access: Record<string, string[]>;
  onChange: (projectId: string, allow: string[]) => void;
  disabled?: boolean;
}

// A search field earns its place once the list no longer fits at a glance.
const SEARCH_FROM = 5;

// One row per project with the count of those given; a project's areas open one project at a time.
export const ProjectAccessList = ({ projects, access, onChange, disabled = false }: ProjectAccessListProps) => {
  const [search, setSearch] = useState("");
  const [customizing, setCustomizing] = useState<string>();
  const query = search.trim().toLowerCase();
  const matches = projects.filter((project) => project.name.toLowerCase().includes(query));
  const granted = projects.filter((project) => (access[project.id] ?? []).length > 0).length;

  return (
    <div>
      <div className="flex min-h-11 items-center gap-3 py-1.5">
        <Microheader className="min-w-0 flex-1 truncate">{`Projects · ${granted} of ${projects.length}`}</Microheader>
        {projects.length >= SEARCH_FROM && (
          <div className="relative w-44 shrink-0">
            <SearchIcon className="pointer-events-none absolute top-1/2 left-2.5 size-3.5 -translate-y-1/2 text-muted-foreground" aria-hidden />
            <Input aria-label="Search projects" placeholder="Search projects" value={search} onChange={(event) => setSearch(event.target.value)} className="h-8 pl-8 text-[13px]" />
          </div>
        )}
      </div>
      {projects.length === 0 && <EmptyRow>No projects to give access to yet.</EmptyRow>}
      {projects.length > 0 && matches.length === 0 && <EmptyRow>No project matches “{search.trim()}”.</EmptyRow>}
      {matches.length > 0 && (
        <ul className="divide-y divide-border border-t border-border">
          {matches.map((project) => (
            <ProjectAccessRow
              key={project.id}
              name={project.name}
              value={access[project.id] ?? []}
              onChange={(allow) => onChange(project.id, allow)}
              customizing={customizing === project.id}
              onCustomize={(open) => setCustomizing(open ? project.id : undefined)}
              disabled={disabled}
            />
          ))}
        </ul>
      )}
    </div>
  );
};
