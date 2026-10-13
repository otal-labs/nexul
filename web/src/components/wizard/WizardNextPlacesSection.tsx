import { ChevronRight, FileText, LayoutDashboard, UserPlus, type LucideIcon } from "lucide-react";

import { useAreaAccess, useVisibleSettingsSections } from "@/hooks/AccessHooks";
import { useFetchProjects } from "@/hooks/ProjectHooks";
import { useWorkspacePath } from "@/hooks/useWorkspacePath";
import { useProjectWizardStore } from "@/stores/projectWizardStore";
import { projectTokenById } from "@/models/Project";

interface WizardNextPlaceRowProps {
  icon: LucideIcon;
  label: string;
  why: string;
  onOpen: () => void;
}

const WizardNextPlaceRow = ({ icon: Icon, label, why, onOpen }: WizardNextPlaceRowProps) => (
  <li>
    <button
      type="button"
      onClick={onOpen}
      className="flex w-full items-center gap-3 px-2 py-3 text-left transition-colors duration-150 ease-standard hover:bg-accent/40"
    >
      <Icon className="size-4 shrink-0 text-muted-foreground" aria-hidden />
      <span className="min-w-0 flex-1">
        <span className="block text-sm font-medium">{label}</span>
        <span className="block truncate text-sm text-muted-foreground">{why}</span>
      </span>
      <ChevronRight className="size-4 shrink-0 text-muted-foreground" aria-hidden />
    </button>
  </li>
);

interface WizardNextPlacesSectionProps {
  // Finishes setup, then opens the path.
  onGo: (path: string) => void;
}

// Where a project without a service goes next, each place shown only to whoever can open it.
export const WizardNextPlacesSection = ({ onGo }: WizardNextPlacesSectionProps) => {
  const projectId = useProjectWizardStore((s) => s.projectId);
  const can = useAreaAccess(projectId ?? undefined);
  const invites = useVisibleSettingsSections()?.includes("team") ?? false;
  const { data: projects } = useFetchProjects();
  const wsPath = useWorkspacePath();
  const token = projectId ? projectTokenById(projects ?? [], projectId) : "";
  if (!can?.("docs") && !can?.("tickets") && !invites) return null;

  return (
    <section aria-labelledby="wizard-next-places" className="space-y-1">
      <h3 id="wizard-next-places" className="text-sm font-medium">
        Where to next
      </h3>
      <ul className="divide-y divide-border border-y border-border">
        {can?.("docs") && (
          <WizardNextPlaceRow icon={FileText} label="Docs" why="Write the brief everyone reads first." onOpen={() => onGo(wsPath("/docs"))} />
        )}
        {can?.("tickets") && (
          <WizardNextPlaceRow icon={LayoutDashboard} label="Board" why="Add the first tickets." onOpen={() => onGo(wsPath(`/board/${token}`))} />
        )}
        {invites && (
          <WizardNextPlaceRow icon={UserPlus} label="Invite people" why="Bring in the client and the team." onOpen={() => onGo("/settings/team")} />
        )}
      </ul>
    </section>
  );
};
