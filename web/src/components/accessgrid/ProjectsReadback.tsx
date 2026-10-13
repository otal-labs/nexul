import { SettingsStatus } from "@/components/settings/SettingsStatus";
import { useAccessDraft } from "@/hooks/useAccessDraft";
import type { AccessGrid } from "@/models/AccessGrid";

interface ProjectsReadbackProps {
  grid: AccessGrid;
}

// The footer's read-back: a project that lists nobody is a signal, never a block.
export const ProjectsReadback = ({ grid }: ProjectsReadbackProps) => {
  const draft = useAccessDraft(grid);
  const empty = grid.projects.filter((project) => {
    const { roleIds, userIds } = draft.listing(project.id);
    return roleIds.length + userIds.length === 0;
  }).length;
  return (
    <span className="min-w-0 flex-1">
      {empty > 0 && (
        <SettingsStatus tone="warning" detail="only Owners see them">
          {empty} {empty === 1 ? "project lists" : "projects list"} nobody yet
        </SettingsStatus>
      )}
      {empty === 0 && <SettingsStatus tone="success">Every project lists someone</SettingsStatus>}
    </span>
  );
};
