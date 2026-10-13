import { Microheader } from "@/components/Microheader";
import { OwnerRow } from "@/components/home/OwnerRow";
import { useFetchWorkspacePeople } from "@/hooks/PeopleHooks";
import type { Workspace } from "@/models/Workspace";

// Who to ask for a role: each Owner opens a direct message, which needs no role to start.
export const WorkspaceOwners = ({ workspace }: { workspace: Workspace }) => {
  const { data: people } = useFetchWorkspacePeople(workspace.id);
  const owners = people?.filter((person) => person.owner) ?? [];
  return (
    <div className="w-72 text-left">
      <Microheader className="text-center">Owners of {workspace.name}</Microheader>
      <ul className="mt-2 divide-y divide-border border-y border-border">
        {owners.map((person) => (
          <OwnerRow key={person.user_id} person={person} workspaceId={workspace.id} />
        ))}
      </ul>
    </div>
  );
};
