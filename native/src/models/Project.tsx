export interface Project {
  id: string;
  name: string;
  /** Immutable 2-5 character uppercase tag (a letter, then letters or digits), unique per workspace, for the human-readable ticket id (PREFIX-N). */
  prefix: string;
}

// No pick in this workspace yet (none, or one from another workspace): the first project stands in.
export const effectiveProject = (projects: Project[] | undefined, selectedId: string | null): Project | undefined =>
  projects?.find((p) => p.id === selectedId) ?? projects?.[0];
