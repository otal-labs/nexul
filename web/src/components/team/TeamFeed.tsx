import { TeamPersonRow } from "@/components/team/TeamPersonRow";
import type { TeamPerson } from "@/models/Team";

interface TeamFeedProps {
  people: TeamPerson[];
  onOpen: (id: string) => void;
}

export const TeamFeed = ({ people, onOpen }: TeamFeedProps) => (
  <ul aria-label="Team" className="divide-y divide-border overflow-hidden rounded-md border bg-card shadow-card">
    {people.map((person, index) => (
      <TeamPersonRow key={person.id} person={person} index={index} onOpen={onOpen} />
    ))}
  </ul>
);
