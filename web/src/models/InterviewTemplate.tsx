export interface InterviewTemplate {
  workspace_id: string;
  /** Markdown a new project's interview memory is copied from once. */
  body: string;
  /** The seeded categories, for resetting the template. */
  default_body: string;
  updated_by: string;
  updated_at: string;
}
