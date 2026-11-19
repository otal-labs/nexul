import { useState } from "react";
import { Lock } from "lucide-react";

import { EmptyRow } from "@/components/EmptyRow";
import { EnterList } from "@/components/EnterList";
import { ErrorDisplay } from "@/components/ErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { Microheader } from "@/components/Microheader";
import { CreatedInvitationLink } from "@/components/member/CreatedInvitationLink";
import { CreateInvitationDialog } from "@/components/member/CreateInvitationDialog";
import { RoleFindField } from "@/components/role/RoleFindField";
import { RoleFoldButton } from "@/components/role/RoleFoldButton";
import { RoleAddPeoplePopover } from "@/components/role/RoleAddPeoplePopover";
import { RoleHolderRow } from "@/components/role/RoleHolderRow";
import { RoleInvitationsSection } from "@/components/role/RoleInvitationsSection";
import { TeamPersonDialog } from "@/components/team/TeamPersonDialog";
import { useOpenRole } from "@/hooks/RoleHooks";
import { useFetchTeam } from "@/hooks/TeamHooks";
import type { CreatedInvitation } from "@/models/Invitation";
import { roleHolders } from "@/models/Team";
import { useWorkspaceStore } from "@/stores/workspaceStore";
import { personName } from "@/utils/TeamUtility";

// Past this many holders a find field leads the list and the rest wait behind Show all.
const FIND_FROM = 8;

// Who holds the open role, with the ways to give it: to someone here, or by a link to someone new.
export const RoleHoldersSection = () => {
  const workspaceId = useWorkspaceStore((s) => s.selectedWorkspaceId);
  const { role, givable, reason } = useOpenRole();
  const { data: team, error, isPending } = useFetchTeam();
  const [created, setCreated] = useState<CreatedInvitation | null>(null);
  const [query, setQuery] = useState("");
  const [all, setAll] = useState(false);
  const [openPerson, setOpenPerson] = useState<string | null>(null);
  const holders = team && role ? roleHolders(team, workspaceId, role.id).sort((a, b) => personName(a).localeCompare(personName(b))) : undefined;
  const changeable = role !== undefined && !role.is_owner_role;
  const needle = query.trim().toLowerCase();
  const matched = holders?.filter((person) => !needle || `${personName(person)} ${person.login}`.toLowerCase().includes(needle));
  // A search shows every match; otherwise a long list stops at its first rows, so what the role allows stays in reach.
  const capped = !all && !needle && (matched?.length ?? 0) > FIND_FROM;
  const shown = capped ? matched?.slice(0, FIND_FROM) : matched;

  return (
    <section className="space-y-3" aria-labelledby="role-people-title">
      {/* From 1024px the heading, its actions and the find field pin while the holders are in view: under the role's header where it pins, else at the top. */}
      <div className="sticky-crumbs band-edge -mx-4 mb-0! px-4 max-lg:relative! sm:-mr-3.5 sm:-ml-6 sm:pr-3.5 sm:pl-6 short-lg:-mt-3 short-lg:pt-3 pin:top-(--role-header)! pin:-mt-5 pin:pt-5">
        {/* The list's top hairline, here so a row passing under the band meets an edge. */}
        <div className="space-y-3 border-b border-border pb-3">
          {/* The field sits with the actions, so a count changing width as you type never moves it; the line wraps before anything overlaps. */}
          <div className="flex min-h-8 flex-wrap items-center gap-x-3 gap-y-2">
            <span className="flex shrink-0 items-center gap-2">
              <span className="flex items-baseline gap-2">
                <Microheader id="role-people-title">People</Microheader>
                {holders && holders.length > 0 && !(needle && matched?.length === 0) && (
                  <span className="font-mono text-micro text-muted-foreground tabular-nums">{needle && matched ? `${matched.length} of ${holders.length}` : holders.length}</span>
                )}
              </span>
              {all && !needle && holders && holders.length > FIND_FROM && <RoleFoldButton open onToggle={() => setAll(false)} label="Show fewer" className="h-6" />}
            </span>
            <div className="flex flex-1 items-center justify-end gap-2">
              {holders && holders.length > FIND_FROM && <RoleFindField value={query} onChange={setQuery} placeholder="Search" label="Find a person" className="w-72 min-w-32 shrink" />}
              {changeable && givable && (
                <>
                  <RoleAddPeoplePopover />
                  <CreateInvitationDialog role={role} onCreated={setCreated} />
                </>
              )}
            </div>
          </div>
          {changeable && !givable && reason && (
            <p className="flex items-start gap-1.5 text-xs text-muted-foreground">
              <Lock aria-hidden className="mt-px size-3 shrink-0" />
              {reason}
            </p>
          )}
        </div>
      </div>
      {created && <CreatedInvitationLink key={created.id} invitation={created} />}
      {isPending && <LoadingDisplay />}
      {error && <ErrorDisplay error={error} />}
      {holders && holders.length === 0 && <EmptyRow flush className="flex min-h-14 items-center border-b border-border">Nobody holds {role?.name} yet.</EmptyRow>}
      {matched && holders && holders.length > 0 && matched.length === 0 && (
        <p role="status" className="flex min-h-14 items-center border-b border-border text-sm text-muted-foreground">
          Nobody holding {role?.name} matches &ldquo;{query.trim()}&rdquo;.
        </p>
      )}
      {shown && shown.length > 0 && (
        <EnterList aria-label={`People holding ${role?.name}`}>
          {shown.map((person) => (
            <RoleHolderRow key={person.id} person={person} onOpen={setOpenPerson} />
          ))}
        </EnterList>
      )}
      {capped && holders && (
        <RoleFoldButton open={false} onToggle={() => setAll(true)} label={`Show all ${holders.length}`} className="-ml-2" />
      )}
      {!needle && <RoleInvitationsSection />}
      <TeamPersonDialog personId={openPerson} onClose={() => setOpenPerson(null)} />
    </section>
  );
};
