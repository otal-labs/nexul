import { HeaderFacts } from "@/components/HeaderFacts";
import { PageHeader } from "@/components/PageHeader";
import { useAreaAccess } from "@/hooks/AccessHooks";
import { useFetchUnreadCount } from "@/hooks/NotificationHooks";
import { useTicketsOnYou } from "@/hooks/useTicketsOnYou";
import { useSelectedWorkspace } from "@/hooks/WorkspaceHooks";
import type { Project } from "@/models/Project";

const Count = ({ n, label }: { n: number; label: string }) => (
  <>
    <span className="font-mono tabular-nums">{n}</span> {label}
  </>
);

// The facts the sections below read from: which project, how much waits on the viewer there, how much is unread (nothing at zero, beside read rows).
export const WorkspaceOverviewHeader = ({ project }: { project: Project | undefined }) => {
  const workspace = useSelectedWorkspace();
  const can = useAreaAccess(project?.id);
  const { data: unread } = useFetchUnreadCount();
  const { data: mine } = useTicketsOnYou(project && can?.("tickets") ? project.id : "");
  return (
    <PageHeader
      title={workspace?.name ?? "Home"}
      meta={
        <HeaderFacts
          parts={[
            project?.name,
            mine && <Count n={mine.length} label={mine.length === 1 ? "ticket on you" : "tickets on you"} />,
            unread && unread.count > 0 && <Count n={unread.count} label="unread" />,
          ]}
        />
      }
    />
  );
};
