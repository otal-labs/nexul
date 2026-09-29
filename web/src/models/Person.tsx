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

// Someone the directory doesn't know (a former member): shown by whatever id or login the caller holds.
export const unknownPerson = (key: string): Person => ({ user_id: key, login: key, display_name: "", avatar_url: "" });

// The one rule for naming a person: their display name, else their login.
export const personLabel = (person: Person): string => person.display_name || person.login;
