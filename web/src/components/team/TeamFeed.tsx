import { EnterList } from "@/components/EnterList";
import { TeamPersonRow } from "@/components/team/TeamPersonRow";
import type { TeamPerson } from "@/models/Team";

interface TeamFeedProps {
  people: TeamPerson[];
  onOpen: (id: string) => void;
}

export const TeamFeed = ({ people, onOpen }: TeamFeedProps) => (
  <EnterList aria-label="Team" className="divide-y divide-border overflow-hidden rounded-md border">
    {people.map((person) => (
      <TeamPersonRow key={person.id} person={person} onOpen={onOpen} />
    ))}
  </EnterList>
);
