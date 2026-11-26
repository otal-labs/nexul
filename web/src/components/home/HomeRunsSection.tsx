import { EmptyRow } from "@/components/EmptyRow";
import { ErrorDisplay } from "@/components/ErrorDisplay";
import { ticketRuns } from "@/components/home/homeRuns";
import { HomeRowsSkeleton } from "@/components/home/HomeRowsSkeleton";
import { HomeRunRow } from "@/components/home/HomeRunRow";
import { HomeSection } from "@/components/home/HomeSection";
import { useFetchTicketsByProject } from "@/hooks/TicketHooks";
import { useFetchActiveTrails } from "@/hooks/TrailHooks";
import { useWorkspacePath } from "@/hooks/useWorkspacePath";
import { boardPath, type Project } from "@/models/Project";

const SHOWN = 5;

export const HomeRunsSection = ({ project }: { project: Project }) => {
  const wsPath = useWorkspacePath();
  const targets = useFetchActiveTrails(project.id);
  const tickets = useFetchTicketsByProject(project.id);
  const error = targets.error ?? tickets.error;
  const runs = targets.data && tickets.data ? ticketRuns(targets.data, tickets.data) : undefined;
  return (
    <HomeSection title="Plays running" more={{ to: wsPath(boardPath(project)), label: "Open board" }}>
      {!error && !runs && <HomeRowsSkeleton label="Loading plays" />}
      {error && <ErrorDisplay error={error} className="p-6" />}
      {runs && runs.length === 0 && <EmptyRow flush className="py-2">No plays are running on tickets in {project.name}.</EmptyRow>}
      {runs && runs.length > 0 && (
        <ul>
          {runs.slice(0, SHOWN).map((run) => (
            <HomeRunRow key={run.ticket.id} run={run} prefix={project.prefix} />
          ))}
        </ul>
      )}
    </HomeSection>
  );
};
