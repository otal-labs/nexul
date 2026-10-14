import { useFetchMe } from "@/hooks/AuthHooks";
import { useFetchTeam } from "@/hooks/TeamHooks";
import { personName } from "@/utils/TeamUtility";

// Names an account from the Team the viewer already holds; someone outside the viewer's view stays unnamed.
export const usePersonNamer = () => {
  const { data: team } = useFetchTeam();
  const { data: me } = useFetchMe();
  return (id: string): string => {
    const person = team?.people.find((candidate) => candidate.id === id);
    if (person) return personName(person);
    if (id === me?.user.id) return "you";
    return "another member";
  };
};
