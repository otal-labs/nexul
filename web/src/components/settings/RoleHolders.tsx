import { PersonAvatar } from "@/components/PersonAvatar";
import type { TeamPerson } from "@/models/Team";

interface RoleHoldersProps {
  people: TeamPerson[];
}

const SHOWN = 4;

// Who holds the role, as overlapping avatars; the count says the rest.
export const RoleHolders = ({ people }: RoleHoldersProps) => {
  const shown = people.slice(0, SHOWN);
  const label = people.length === 1 ? "1 person" : `${people.length} people`;
  return (
    <span className="inline-flex shrink-0 items-center gap-2" title={people.map((p) => p.display_name || p.login).join(", ")}>
      {shown.length > 0 && (
        <span className="flex -space-x-1.5">
          {shown.map((person) => (
            <PersonAvatar
              key={person.id}
              login={person.login}
              src={person.avatar_url}
              label={person.display_name || person.login}
              className="size-6 text-[10px] ring-2 ring-card"
            />
          ))}
        </span>
      )}
      <span className="font-mono text-xs tabular-nums text-muted-foreground">{people.length === 0 ? "nobody" : label}</span>
    </span>
  );
};
