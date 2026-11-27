import { SparklesIcon } from "lucide-react";

import { PlayTreeHeader } from "@/components/agent/PlayTreeHeader";
import { PlayTreeSection } from "@/components/agent/PlayTreeSection";
import { NewPlayButton } from "@/components/agent/NewPlayButton";
import { EmptyState } from "@/components/EmptyState";
import { ErrorDisplay } from "@/components/ErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { useFetchWorkspacePlays } from "@/hooks/PlayHooks";
import { AGENT_TAB_LABELS, type AgentTab } from "@/models/Agent";
import { useWorkspaceStore } from "@/stores/workspaceStore";

// The plays of one type as a tree; the page's count and this list read the same query.
export const PlayTabFeed = ({ tab }: { tab: AgentTab }) => {
  const workspaceId = useWorkspaceStore((s) => s.selectedWorkspaceId);
  const { data, error, isPending } = useFetchWorkspacePlays(workspaceId);
  const plays = data?.filter((play) => play.type === tab);

  return (
    <div className={plays && plays.length > 0 ? "border-b border-border" : undefined}>
      {isPending && <LoadingDisplay label="Loading plays…" />}
      {error && <ErrorDisplay error={error} title="Couldn't load plays." />}
      {plays && plays.length === 0 && (
        <EmptyState role="status" icon={SparklesIcon} title={`No ${AGENT_TAB_LABELS[tab].toLowerCase()} yet`} action={<NewPlayButton tab={tab} />} />
      )}
      {plays && plays.length > 0 && <PlayTreeHeader />}
      {plays?.map((play) => <PlayTreeSection key={play.id} play={play} />)}
    </div>
  );
};
