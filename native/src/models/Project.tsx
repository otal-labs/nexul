export interface Project {
  id: string;
  name: string;
  /** Immutable 2-5 uppercase-letter tag, unique per workspace, for the human-readable ticket id (PREFIX-N). */
  prefix: string;
}
