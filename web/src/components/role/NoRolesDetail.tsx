import { Container } from "@/components/Container";
import { EmptyRow } from "@/components/EmptyRow";
import { EnterList } from "@/components/EnterList";
import { ErrorDisplay } from "@/components/ErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { RoleDetailHeader } from "@/components/role/RoleDetailHeader";
import { RoleHolderRow } from "@/components/role/RoleHolderRow";
import { useFetchTeam } from "@/hooks/TeamHooks";
import { useSelectedWorkspace } from "@/hooks/WorkspaceHooks";
import { roleless } from "@/models/Team";
import { personName } from "@/utils/TeamUtility";

// The members holding no role, who see no project or play here until a role is given from its own page.
export const NoRolesDetail = () => {
  const workspace = useSelectedWorkspace();
  const { data: team, error, isPending } = useFetchTeam();
  const people = team && workspace ? roleless(team, workspace.id).sort((a, b) => personName(a).localeCompare(personName(b))) : undefined;
  const name = workspace?.name ?? "this workspace";

  return (
    <Container className="space-y-6 pt-0 pb-6 sm:pr-3.5">
      <RoleDetailHeader title="No roles" line={`In ${name} but holding no role, so they see no projects or plays.`} />
      {isPending && <LoadingDisplay />}
      {error && <ErrorDisplay error={error} />}
      {people && people.length === 0 && <EmptyRow flush>Everyone in {name} holds a role.</EmptyRow>}
      {people && people.length > 0 && (
        <EnterList aria-label="People holding no role" className="border-t border-border">
          {people.map((person) => (
            <RoleHolderRow key={person.id} person={person} />
          ))}
        </EnterList>
      )}
    </Container>
  );
};
