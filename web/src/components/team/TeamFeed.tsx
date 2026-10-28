import { EnterList } from "@/components/EnterList";
import { settingsListClass } from "@/components/settings/SettingsRow";
import { TeamPersonRow } from "@/components/team/TeamPersonRow";
import type { TeamPerson } from "@/models/Team";

interface TeamFeedProps {
  people: TeamPerson[];
  onOpen: (id: string) => void;
}

export const TeamFeed = ({ people, onOpen }: TeamFeedProps) => (
  <EnterList aria-label="Team" className={settingsListClass}>
    {people.map((person) => (
      <TeamPersonRow key={person.id} person={person} onOpen={onOpen} />
    ))}
  </EnterList>
);
