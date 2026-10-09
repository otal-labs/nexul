// Mirrors internal/tenancy/model.go's Person: all any member of a workspace sees of another.
export interface Person {
  user_id: string;
  login: string;
  // The name they chose, else their sign-in account's name; empty when neither exists.
  display_name: string;
  // An https URL, or a /api/people/... path that needs the session to load.
  avatar_url: string;
}

export interface PeopleList {
  people: Person[];
}

const unknownPeople = new Map<string, Person>();

// Someone the directory doesn't know (a former member, a bot): shown by whatever id or login the caller holds.
// One object per key, so a memoized row authored by them doesn't re-render on every lookup.
export const unknownPerson = (key: string): Person => {
  const known = unknownPeople.get(key);
  if (known) return known;
  const person = { user_id: key, login: key, display_name: "", avatar_url: "" };
  unknownPeople.set(key, person);
  return person;
};

// The one rule for naming a person: their display name, else their login.
export const personLabel = (person: Person): string => person.display_name || person.login;
