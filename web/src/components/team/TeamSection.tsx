import { useState } from "react";
import { useLocation, useNavigate, useSearchParams } from "react-router";

import { ErrorDisplay } from "@/components/ErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { EmptyRow } from "@/components/EmptyRow";
import { PageTabs, PageTabsContent } from "@/components/PageTabs";
import { CreatedInvitationLink } from "@/components/member/CreatedInvitationLink";
import { CreateInvitationDialog } from "@/components/member/CreateInvitationDialog";
import { InvitationsFeed } from "@/components/member/InvitationsFeed";
import { SettingsCard } from "@/components/settings/SettingsCard";
import { TeamFeed } from "@/components/team/TeamFeed";
import { TeamPersonDialog } from "@/components/team/TeamPersonDialog";
import { useHasInstancePermission } from "@/hooks/AccessHooks";
import { useFetchTeam } from "@/hooks/TeamHooks";
import { useTabPath } from "@/hooks/useTabPath";
import type { CreatedInvitation } from "@/models/Invitation";

// The open person lives in ?person= so a detail is linkable and survives a refresh; a new link opens the Invitations tab, shown on top so it isn't missed.
export const TeamSection = () => {
  const { data: team, isPending, error } = useFetchTeam();
  const [params, setParams] = useSearchParams();
  const [created, setCreated] = useState<CreatedInvitation | null>(null);
  const navigate = useNavigate();
  const { search } = useLocation();
  const { tabPath } = useTabPath();
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

  const showCreated = (invitation: CreatedInvitation) => {
    setCreated(invitation);
    navigate({ pathname: tabPath("invitations"), search });
  };

  return (
    <SettingsCard
      id="team"
      title="Team"
      description={description}
      footer={<CreateInvitationDialog onCreated={showCreated} />}
    >
      <PageTabs
        label="Team"
        tabs={[
          { value: "people", label: "People" },
          { value: "invitations", label: "Invitations" },
        ]}
      >
        <PageTabsContent value="people">
          {isPending && <LoadingDisplay />}
          {error && <ErrorDisplay error={error} />}
          {team && team.people.length === 0 && <EmptyRow>No one here yet. Invite someone to add them.</EmptyRow>}
          {team && team.people.length > 0 && <TeamFeed people={team.people} onOpen={open} />}
        </PageTabsContent>
        <PageTabsContent value="invitations">
          {created && <CreatedInvitationLink key={created.id} invitation={created} />}
          <InvitationsFeed />
        </PageTabsContent>
      </PageTabs>
      <TeamPersonDialog personId={personId} onClose={() => open(null)} />
    </SettingsCard>
  );
};
