import { useSearchParams } from "react-router";

import { ErrorDisplay } from "@/components/ErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { EmptyRow } from "@/components/EmptyRow";
import { CreateInvitationDialog } from "@/components/member/CreateInvitationDialog";
import { InvitationsFeed } from "@/components/member/InvitationsFeed";
import { SettingsCard } from "@/components/settings/SettingsCard";
import { TeamFeed } from "@/components/team/TeamFeed";
import { TeamPersonDialog } from "@/components/team/TeamPersonDialog";
import { useHasInstancePermission } from "@/hooks/AccessHooks";
import { useFetchTeam } from "@/hooks/TeamHooks";

// The open person lives in ?person= so a detail is linkable and survives a refresh.
export const TeamSection = () => {
  const { data: team, isPending, error } = useFetchTeam();
  const [params, setParams] = useSearchParams();
  const personId = params.get("person");
  const everyone = useHasInstancePermission("accounts:read");
  const scope = everyone ? "Everyone on this instance" : "People in the workspaces you manage";
  const description = `${scope}, and what they can reach in each workspace. Invite someone with a link.`;

  const open = (id: string | null) =>
    setParams(
      (current) => {
        const next = new URLSearchParams(current);
        if (id) next.set("person", id);
        if (!id) next.delete("person");
        return next;
      },
      { replace: true },
    );

  return (
    <SettingsCard
      id="team"
      title="Team"
      description={description}
      footer={<CreateInvitationDialog />}
    >
      {isPending && <LoadingDisplay />}
      {error && <ErrorDisplay error={error} />}
      {team && team.people.length === 0 && <EmptyRow>No one here yet. Invite someone to add them.</EmptyRow>}
      {team && team.people.length > 0 && <TeamFeed people={team.people} onOpen={open} />}
      <InvitationsFeed />
      <TeamPersonDialog personId={personId} onClose={() => open(null)} />
    </SettingsCard>
  );
};
