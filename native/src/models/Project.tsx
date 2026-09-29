export interface Project {
  id: string;
  name: string;
  /** Immutable 2-5 character uppercase tag (a letter, then letters or digits), unique per workspace, for the human-readable ticket id (PREFIX-N). */
  prefix: string;
}
