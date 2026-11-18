import { useState } from "react";
import { Lock } from "lucide-react";

import { EmptyRow } from "@/components/EmptyRow";
import { EnterList } from "@/components/EnterList";
import { ErrorDisplay } from "@/components/ErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { Microheader } from "@/components/Microheader";
import { CreatedInvitationLink } from "@/components/member/CreatedInvitationLink";
import { CreateInvitationDialog } from "@/components/member/CreateInvitationDialog";
import { RoleAddPeoplePopover } from "@/components/role/RoleAddPeoplePopover";
import { RoleHolderRow } from "@/components/role/RoleHolderRow";
import { RoleInvitationsSection } from "@/components/role/RoleInvitationsSection";
import { useOpenRole } from "@/hooks/RoleHooks";
import { useFetchTeam } from "@/hooks/TeamHooks";
import type { CreatedInvitation } from "@/models/Invitation";
import { roleHolders } from "@/models/Team";
import { useWorkspaceStore } from "@/stores/workspaceStore";

// Who holds the open role, with the ways to give it: to someone here, or by a link to someone new.
export const RoleHoldersSection = () => {
  const workspaceId = useWorkspaceStore((s) => s.selectedWorkspaceId);
  const { role, givable, reason } = useOpenRole();
  const { data: team, error, isPending } = useFetchTeam();
  const [created, setCreated] = useState<CreatedInvitation | null>(null);
  const holders = team && role ? roleHolders(team, workspaceId, role.id) : undefined;
  const changeable = role !== undefined && !role.is_owner_role;

  return (
    <section className="space-y-3" aria-labelledby="role-people-title">
      <div className="flex min-h-8 flex-wrap items-center justify-between gap-2">
        <Microheader id="role-people-title">People{holders && ` · ${holders.length}`}</Microheader>
        {changeable && givable && (
          <div className="flex items-center gap-2">
            <RoleAddPeoplePopover />
            <CreateInvitationDialog role={role} onCreated={setCreated} />
          </div>
        )}
      </div>
      {changeable && !givable && reason && (
        <p className="flex items-start gap-1.5 text-xs text-muted-foreground">
          <Lock aria-hidden className="mt-px size-3 shrink-0" />
          {reason}
        </p>
      )}
      {created && <CreatedInvitationLink key={created.id} invitation={created} />}
      {isPending && <LoadingDisplay />}
      {error && <ErrorDisplay error={error} />}
      {holders && holders.length === 0 && <EmptyRow flush>Nobody holds {role?.name} yet.</EmptyRow>}
      {holders && holders.length > 0 && (
        <EnterList aria-label={`People holding ${role?.name}`} className="border-t border-border">
          {holders.map((person) => (
            <RoleHolderRow key={person.id} person={person} />
          ))}
        </EnterList>
      )}
      <RoleInvitationsSection />
    </section>
  );
};
