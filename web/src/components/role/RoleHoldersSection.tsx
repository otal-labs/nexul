import { useState } from "react";
import { Lock, Search, X } from "lucide-react";

import { EmptyRow } from "@/components/EmptyRow";
import { EnterList } from "@/components/EnterList";
import { ErrorDisplay } from "@/components/ErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { Microheader } from "@/components/Microheader";
import { Input } from "@/components/ui/input";
import { CreatedInvitationLink } from "@/components/member/CreatedInvitationLink";
import { CreateInvitationDialog } from "@/components/member/CreateInvitationDialog";
import { RoleFoldButton } from "@/components/role/RoleFoldButton";
import { RoleAddPeoplePopover } from "@/components/role/RoleAddPeoplePopover";
import { RoleHolderRow } from "@/components/role/RoleHolderRow";
import { RoleInvitationsSection } from "@/components/role/RoleInvitationsSection";
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
  const holders = team && role ? roleHolders(team, workspaceId, role.id) : undefined;
  const changeable = role !== undefined && !role.is_owner_role;
  const needle = query.trim().toLowerCase();
  const matched = holders?.filter((person) => !needle || `${personName(person)} ${person.login}`.toLowerCase().includes(needle));
  // A search shows every match; otherwise a long list stops at its first rows, so what the role allows stays in reach.
  const capped = !all && !needle && (matched?.length ?? 0) > FIND_FROM;
  const shown = capped ? matched?.slice(0, FIND_FROM) : matched;

  return (
    <section className="space-y-3" aria-labelledby="role-people-title">
      <div className="flex min-h-8 flex-wrap items-center justify-between gap-2">
        <div className="flex items-baseline gap-2">
          <Microheader id="role-people-title">People</Microheader>
          {holders && holders.length > 0 && (
            <span className="font-mono text-micro text-muted-foreground tabular-nums">{needle && matched ? `${matched.length} of ${holders.length}` : holders.length}</span>
          )}
        </div>
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
      {holders && holders.length > FIND_FROM && (
        <div className="relative max-w-xs">
          <Search aria-hidden className="pointer-events-none absolute top-1/2 left-2.5 size-3.5 -translate-y-1/2 text-muted-foreground" />
          <Input value={query} onChange={(event) => setQuery(event.target.value)} placeholder={`Find among ${holders.length}`} aria-label="Find a person" className="h-8 pr-8 pl-8" />
          {query && (
            <button type="button" aria-label="Clear the search" onClick={() => setQuery("")} className="absolute top-1/2 right-1.5 flex size-6 -translate-y-1/2 items-center justify-center rounded-sm text-muted-foreground hover:bg-accent hover:text-foreground">
              <X aria-hidden className="size-3.5" />
            </button>
          )}
        </div>
      )}
      {matched && holders && holders.length > 0 && matched.length === 0 && (
        <p role="status" className="flex min-h-14 items-center border-y border-border text-sm text-muted-foreground">
          Nobody holding {role?.name} matches &ldquo;{query.trim()}&rdquo;.
        </p>
      )}
      {shown && shown.length > 0 && (
        <EnterList aria-label={`People holding ${role?.name}`} className="border-t border-border">
          {shown.map((person) => (
            <RoleHolderRow key={person.id} person={person} />
          ))}
        </EnterList>
      )}
      {capped && holders && (
        <RoleFoldButton open={false} onToggle={() => setAll(true)} label={`Show all ${holders.length}`} className="-ml-2" />
      )}
      <RoleInvitationsSection />
    </section>
  );
};
