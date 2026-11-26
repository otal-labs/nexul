import { EmptyRow } from "@/components/EmptyRow";
import { ErrorDisplay } from "@/components/ErrorDisplay";
import { HomeRowsSkeleton } from "@/components/home/HomeRowsSkeleton";
import { HomeSection } from "@/components/home/HomeSection";
import { HomeTicketRow } from "@/components/home/HomeTicketRow";
import { useTicketsOnYou } from "@/hooks/useTicketsOnYou";
import { useWorkspacePath } from "@/hooks/useWorkspacePath";
import { boardPath, type Project } from "@/models/Project";

const SHOWN = 8;

export const HomeTicketsSection = ({ project }: { project: Project }) => {
  const wsPath = useWorkspacePath();
  const { data: mine, error, isPending } = useTicketsOnYou(project.id);
  const hidden = (mine?.length ?? 0) - SHOWN;
  return (
    <HomeSection title="Your tickets" more={{ to: wsPath(boardPath(project)), label: "Open board" }}>
      {isPending && <HomeRowsSkeleton label="Loading your tickets" />}
      {error && <ErrorDisplay error={error} className="p-6" />}
      {mine && mine.length === 0 && <EmptyRow flush className="py-2">Nothing is waiting on you in {project.name}.</EmptyRow>}
      {mine && mine.length > 0 && (
        <ul>
          {mine.slice(0, SHOWN).map((item) => (
            <HomeTicketRow key={item.ticket.id} item={item} prefix={project.prefix} />
          ))}
        </ul>
      )}
      {hidden > 0 && (
        <p className="pt-2 text-xs text-muted-foreground">
          <span className="font-mono tabular-nums">{hidden}</span> more on the board
        </p>
      )}
    </HomeSection>
  );
};
