import { useState } from "react";

import { Container } from "@/components/Container";
import { EmptyRow } from "@/components/EmptyRow";
import { EnterList } from "@/components/EnterList";
import { ErrorDisplay } from "@/components/ErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { RoleDetailHeader } from "@/components/role/RoleDetailHeader";
import { RoleFindField } from "@/components/role/RoleFindField";
import { RoleHolderRow } from "@/components/role/RoleHolderRow";
import { TeamPersonDialog } from "@/components/team/TeamPersonDialog";
import { useFetchTeam } from "@/hooks/TeamHooks";
import { useSelectedWorkspace } from "@/hooks/WorkspaceHooks";
import { roleless } from "@/models/Team";
import { personName } from "@/utils/TeamUtility";

// Past this many, a find field leads the list, as on a role's page.
const FIND_FROM = 8;

// The members holding no role, who see no project or play here until a role is given from its own page.
export const NoRolesDetail = () => {
  const workspace = useSelectedWorkspace();
  const { data: team, error, isPending } = useFetchTeam();
  const [openPerson, setOpenPerson] = useState<string | null>(null);
  const [query, setQuery] = useState("");
  const people = team && workspace ? roleless(team, workspace.id).sort((a, b) => personName(a).localeCompare(personName(b))) : undefined;
  const name = workspace?.name ?? "this workspace";
  const needle = query.trim().toLowerCase();
  const shown = people?.filter((person) => !needle || `${personName(person)} ${person.login}`.toLowerCase().includes(needle));

  return (
    <Container className="space-y-5 pt-0 pb-6 sm:pr-3.5">
      <RoleDetailHeader title="No roles" line={`In ${name} but holding no role, so they see no projects or plays.`} />
      {isPending && <LoadingDisplay />}
      {error && <ErrorDisplay error={error} />}
      {people && people.length === 0 && <EmptyRow flush>Everyone in {name} holds a role.</EmptyRow>}
      {people && people.length > 0 && shown && (
        <section className="space-y-3" aria-labelledby="no-roles-people">
          <div className="flex min-h-8 items-center gap-2">
            <h3 id="no-roles-people" className="text-sm font-semibold">People</h3>
            <span className="font-mono text-micro text-muted-foreground tabular-nums">{needle ? `${shown.length} of ${people.length}` : people.length}</span>
          </div>
          {people.length > FIND_FROM && <RoleFindField value={query} onChange={setQuery} placeholder="Search" label="Find a person" className="max-w-xs" />}
          {shown.length === 0 && (
            <p role="status" className="flex min-h-14 items-center border-y border-border text-sm text-muted-foreground">
              Nobody here matches &ldquo;{query.trim()}&rdquo;.
            </p>
          )}
          {shown.length > 0 && (
            <EnterList aria-label="People holding no role" className="border-t border-border">
              {shown.map((person) => (
                <RoleHolderRow key={person.id} person={person} onOpen={setOpenPerson} />
              ))}
            </EnterList>
          )}
        </section>
      )}
      <TeamPersonDialog personId={openPerson} onClose={() => setOpenPerson(null)} />
    </Container>
  );
};
