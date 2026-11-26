import { useFetchMe } from "@/hooks/AuthHooks";
import { useFetchTeam } from "@/hooks/TeamHooks";
import { personName } from "@/utils/TeamUtility";

// Names an account, by id or login, from the Team the viewer already holds; someone outside the viewer's view stays unnamed.
export const usePersonNamer = () => {
  const { data: team } = useFetchTeam();
  const { data: me } = useFetchMe();
  return (id: string): string => {
    const person = team?.people.find((candidate) => candidate.id === id || candidate.login === id);
    if (person) return personName(person);
    if (id === me?.user.id) return "you";
    return "another member";
  };
};

// Who started a deploy or job: an account id (with ":mcp" when an agent acted for it) by name, any other word as written.
export const useActorNamer = () => {
  const nameOf = usePersonNamer();
  return (by: string | undefined): string | undefined => {
    if (!by) return undefined;
    const id = by.replace(/:mcp$/, "");
    const name = nameOf(id);
    const who = name === "another member" && !/^[0-9a-f]{8}-[0-9a-f]{4}-/i.test(id) ? id : name;
    return by.endsWith(":mcp") ? `${who} via MCP` : who;
  };
};
